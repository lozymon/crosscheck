package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"

	"github.com/lozymon/crosscheck/adapters/dynamodb"
	"github.com/lozymon/crosscheck/adapters/lambda"
	"github.com/lozymon/crosscheck/adapters/mongodb"
	"github.com/lozymon/crosscheck/adapters/mysql"
	"github.com/lozymon/crosscheck/adapters/redis"
	s3adapter "github.com/lozymon/crosscheck/adapters/s3"
	"github.com/lozymon/crosscheck/adapters/sns"
	"github.com/lozymon/crosscheck/adapters/sqs"
	"github.com/lozymon/crosscheck/config"
	"github.com/lozymon/crosscheck/discovery"
	"github.com/lozymon/crosscheck/env"
	"github.com/lozymon/crosscheck/httpclient"
	"github.com/lozymon/crosscheck/reporter"
	"github.com/lozymon/crosscheck/runner"
)

var (
	runEnvFile    string
	runEnvVars    []string
	runFilter     string
	runInsecure   bool
	runOutputFile string
	runReporter   string
	runWatch      bool
)

var runCmd = &cobra.Command{
	Use:   "run [file or directory]",
	Short: "Run test files",
	Long: `Run crosscheck test files (*.cx.yaml).

If no path is given, recursively finds all *.cx.yaml files in the current directory.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "."

		if len(args) == 1 {
			path = args[0]
		}

		if runWatch {
			return watchAndRun(cmd, path)
		}

		return runTests(cmd, path)
	},
}

func init() {
	runCmd.Flags().StringVar(&runEnvFile, "env-file", ".env", "Path to .env file")
	runCmd.Flags().StringArrayVar(&runEnvVars, "env", nil, "Override env variables (KEY=VALUE)")
	runCmd.Flags().StringVar(&runFilter, "filter", "", "Run only tests matching pattern (e.g. 'order*')")
	runCmd.Flags().BoolVar(&runInsecure, "insecure", false, "Skip TLS certificate verification")
	runCmd.Flags().StringVar(&runOutputFile, "output-file", "", "Write JSON results to file")
	runCmd.Flags().StringVar(&runReporter, "reporter", "pretty", "Reporter format: pretty, json, junit")
	runCmd.Flags().BoolVar(&runWatch, "watch", false, "Watch for file changes and re-run (Phase 2)")
}

func runTests(cmd *cobra.Command, path string) error {
	// Discover test files.
	files, err := discovery.Find(path)

	if err != nil {
		return &ExitError{Code: ExitConfigError, Message: err.Error()}
	}

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "no *.cx.yaml test files found")

		return nil
	}

	// Build shared dependencies used across all files.
	client := httpclient.New(runInsecure)

	rep, err := reporter.New(runReporter, os.Stdout)

	if err != nil {
		return &ExitError{Code: ExitConfigError, Message: err.Error()}
	}

	defer func() {
		if closeErr := rep.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "reporter close: %v\n", closeErr)
		}
	}()

	// Connect optional adapters from environment.
	opts := runner.Options{}

	// AWS adapters — all share one config loaded from the default credential chain.
	// Activated when AWS_REGION is set; credentials come from env, profile, or instance role.
	if awsRegion := os.Getenv("AWS_REGION"); awsRegion != "" {
		awsCfg, awsErr := awsconfig.LoadDefaultConfig(cmd.Context())

		if awsErr != nil {
			return &ExitError{Code: ExitConnectError, Message: fmt.Sprintf("aws config: %v", awsErr)}
		}

		opts.SQS = sqs.New(awsCfg)
		opts.SNS = sns.New(awsCfg)
		opts.S3 = s3adapter.New(awsCfg)
		opts.DynamoDB = dynamodb.New(awsCfg)
		opts.Lambda = lambda.New(awsCfg)
	}

	if mongoURL := os.Getenv("MONGODB_URL"); mongoURL != "" {
		mongoAdapter, mongoErr := mongodb.New(cmd.Context(), mongoURL)

		if mongoErr != nil {
			return &ExitError{Code: ExitConnectError, Message: mongoErr.Error()}
		}

		defer func() { _ = mongoAdapter.Close(context.Background()) }()

		opts.MongoDB = mongoAdapter
	}

	if mysqlURL := os.Getenv("MYSQL_URL"); mysqlURL != "" {
		mysqlAdapter, mysqlErr := mysql.New(cmd.Context(), mysqlURL)

		if mysqlErr != nil {
			return &ExitError{Code: ExitConnectError, Message: mysqlErr.Error()}
		}

		defer func() { _ = mysqlAdapter.Close() }()

		opts.MySQL = mysqlAdapter
	}

	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		redisAdapter, redisErr := redis.New(cmd.Context(), redisURL)

		if redisErr != nil {
			return &ExitError{Code: ExitConnectError, Message: redisErr.Error()}
		}

		defer func() { _ = redisAdapter.Close() }()

		opts.Redis = redisAdapter
	}

	var (
		totalFailed   int
		allResults    []*runner.FileResult
		anyConnectErr error
	)

	// Run each file.
	for _, file := range files {
		tf, parseErr := config.Parse(file)

		if parseErr != nil {
			return &ExitError{Code: ExitConfigError, Message: parseErr.Error()}
		}

		// Apply --filter: keep only tests whose names match the glob pattern.
		if runFilter != "" {
			tf.Tests = filterTests(tf.Tests, runFilter)
		}

		vars := env.Load(runEnvFile, runEnvVars, tf.Env)

		result := runner.RunFile(cmd.Context(), tf, vars, client, opts)

		if writeErr := rep.Write(result); writeErr != nil {
			fmt.Fprintf(os.Stderr, "reporter error: %v\n", writeErr)
		}

		totalFailed += result.Failed
		allResults = append(allResults, result)

		if result.SetupErr != nil {
			anyConnectErr = result.SetupErr
		}
	}

	// Write combined JSON output file if requested.
	if runOutputFile != "" && len(allResults) > 0 {
		// Write the last (or only) result; for multi-file runs a single file
		// gets the last suite. A merged format is Phase 2.
		if writeErr := reporter.WriteJSONFile(runOutputFile, allResults[len(allResults)-1]); writeErr != nil {
			fmt.Fprintf(os.Stderr, "output-file error: %v\n", writeErr)
		}
	}

	// Determine exit code across all files.
	if anyConnectErr != nil {
		return &ExitError{Code: ExitConnectError, Message: anyConnectErr.Error()}
	}

	if totalFailed > 0 {
		return &ExitError{Code: ExitTestFailure}
	}

	return nil
}

// filterTests returns only tests whose Name matches the glob pattern.
// Tests with names that fail to match (or cause a pattern error) are excluded.
func filterTests(tests []config.Test, pattern string) []config.Test {
	var out []config.Test

	for _, t := range tests {
		matched, err := filepath.Match(pattern, t.Name)

		if err == nil && matched {
			out = append(out, t)
		}
	}

	return out
}
