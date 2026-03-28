package config

// TestFile represents a *.cx.yaml test file.
type TestFile struct {
	Version     int               `yaml:"version"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Env         map[string]string `yaml:"env"`
	Auth        *Auth             `yaml:"auth"`
	Mock        *MockServer       `yaml:"mock"`
	Setup       []Hook            `yaml:"setup"`
	Teardown    []Hook            `yaml:"teardown"`
	Tests       []Test            `yaml:"tests"`
}

// MockServer configures the built-in outbound-call capture server.
// When present, crosscheck starts a local HTTP server before the tests run
// and injects its base URL as MOCK_URL into the variable namespace.
// Tests can point their application at MOCK_URL and later assert that the
// expected requests were received using `adapter: mock` in the services block.
type MockServer struct {
	// Port to listen on. 0 (or omitted) lets the OS pick a free port.
	Port int `yaml:"port"`
}

// Auth defines how to authenticate before running tests.
type Auth struct {
	Type    string     `yaml:"type"` // "login" or "static"
	Request *Request   `yaml:"request"`
	Capture CaptureMap `yaml:"capture"`
	Inject  AuthInject `yaml:"inject"`
}

type AuthInject struct {
	Header string `yaml:"header"`
	Format string `yaml:"format"`
}

// Test is a single test step.
type Test struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Timeout     string          `yaml:"timeout"`
	Retry       int             `yaml:"retry"`
	RetryDelay  string          `yaml:"retry_delay"`
	Setup       []Hook          `yaml:"setup"`
	Teardown    []Hook          `yaml:"teardown"`
	Request     *Request        `yaml:"request"`
	Response    *ResponseAssert `yaml:"response"`
	Database    []DBAssert      `yaml:"database"`
	Services    []ServiceAssert `yaml:"services"`
}

// Request defines an HTTP request.
type Request struct {
	Method  string            `yaml:"method"`
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers"`
	Body    any               `yaml:"body"`
}

// ResponseAssert defines expected HTTP response assertions.
type ResponseAssert struct {
	Status  int               `yaml:"status"`
	Headers map[string]string `yaml:"headers"`
	Body    any               `yaml:"body"`
}

// DBAssert defines a database assertion after a request.
type DBAssert struct {
	Adapter string           `yaml:"adapter"`
	Query   string           `yaml:"query"`
	Params  map[string]any   `yaml:"params"`
	WaitFor *WaitFor         `yaml:"wait_for"`
	Expect  []map[string]any `yaml:"expect"`
}

// ServiceAssert defines a service (Redis, SQS, SNS, S3, DynamoDB, Lambda, mock) assertion.
type ServiceAssert struct {
	Adapter     string         `yaml:"adapter"`
	Key         string         `yaml:"key"`           // Redis key | S3 object key | Lambda function name | DynamoDB partition key value
	KeyName     string         `yaml:"key_name"`      // DynamoDB: partition key attribute name (default: "id")
	SortKey     string         `yaml:"sort_key"`      // DynamoDB: sort key value (optional)
	SortKeyName string         `yaml:"sort_key_name"` // DynamoDB: sort key attribute name
	Queue       string         `yaml:"queue"`         // SQS queue URL | SNS: SQS queue URL subscribed to the topic
	Bucket      string         `yaml:"bucket"`        // S3 bucket name
	Table       string         `yaml:"table"`         // DynamoDB table name
	Payload     map[string]any `yaml:"payload"`       // Lambda: invocation input payload
	Path        string         `yaml:"path"`          // mock: URL path filter (e.g. "/webhook")
	Method      string         `yaml:"method"`        // mock: HTTP method filter (e.g. "POST"); empty = any
	WaitFor     *WaitFor       `yaml:"wait_for"`      // poll until assertion passes (async flows)
	Expect      map[string]any `yaml:"expect"`
}

// WaitFor defines polling behaviour for async assertions.
type WaitFor struct {
	Timeout  string `yaml:"timeout"`
	Interval string `yaml:"interval"`
}

// Hook is a shell command run in setup or teardown.
type Hook struct {
	Run string `yaml:"run"`
}

// CaptureMap maps capture variable names to JSONPath expressions.
type CaptureMap map[string]string
