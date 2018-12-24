package deployment

// Deployer is an interface that can be used to install the secrets client
// and copy a service configuration to a target machine.
type Deployer interface {
	Configure([]byte) error
}
