package config

// TestFile represents a *.cx.yaml test file.
type TestFile struct {
	Version     int               `yaml:"version"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Env         map[string]string `yaml:"env"`
	Auth        *Auth             `yaml:"auth"`
	Setup       []Hook            `yaml:"setup"`
	Teardown    []Hook            `yaml:"teardown"`
	Tests       []Test            `yaml:"tests"`
}

// Auth defines how to authenticate before running tests.
type Auth struct {
	Type    string      `yaml:"type"` // "login" or "static"
	Request *Request    `yaml:"request"`
	Capture CaptureMap  `yaml:"capture"`
	Inject  AuthInject  `yaml:"inject"`
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
	Status  int            `yaml:"status"`
	Headers map[string]string `yaml:"headers"`
	Body    any            `yaml:"body"`
}

// DBAssert defines a database assertion after a request.
type DBAssert struct {
	Adapter string            `yaml:"adapter"`
	Query   string            `yaml:"query"`
	Params  map[string]any    `yaml:"params"`
	WaitFor *WaitFor          `yaml:"wait_for"`
	Expect  []map[string]any  `yaml:"expect"`
}

// ServiceAssert defines a service (Redis, SQS, etc.) assertion.
type ServiceAssert struct {
	Adapter string         `yaml:"adapter"`
	Key     string         `yaml:"key"`   // Redis key
	Queue   string         `yaml:"queue"` // SQS queue name
	Expect  map[string]any `yaml:"expect"`
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
