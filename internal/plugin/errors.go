package plugin

import "errors"

var (
	ErrPluginAlreadyRegistered = errors.New("plugin already registered")
	ErrPluginNotFound          = errors.New("plugin not found")
	ErrPluginProcessingFailed  = errors.New("plugin processing failed")
	ErrNoPluginsEnabled        = errors.New("no plugins enabled")
	ErrPluginNotImplemented    = errors.New("plugin does not implement")
)

// PluginAlreadyRegisteredError returns an error for duplicate plugin registration
func PluginAlreadyRegisteredError(name string) error {
	return errors.New("plugin already registered: " + name)
}

// PluginNotFoundError returns an error when plugin is not found
func PluginNotFoundError(name string) error {
	return errors.New("plugin not found: " + name)
}

// PluginProcessingFailedError returns an error when plugin processing fails
func PluginProcessingFailedError(name string, err error) error {
	return errors.New("plugin processing failed (" + name + "): " + err.Error())
}

func NoPluginsEnabledError() error {
	return ErrNoPluginsEnabled
}
