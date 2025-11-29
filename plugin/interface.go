package plugin

import (
	"encoding/json"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

// StatePlugin is the interface that we're exposing as a plugin.
// This will be implemented by server block plugins and consumed by livepage.
type StatePlugin interface {
	// Change executes an action on the state
	Change(action string, data map[string]interface{}) error

	// GetState returns the current state as JSON for rendering
	GetState() (json.RawMessage, error)
}

// Handshake is a common handshake that is shared between plugin and host.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "LIVEPAGE_PLUGIN",
	MagicCookieValue: "state",
}

// PluginMap is the map of plugins we can dispense.
var PluginMap = map[string]plugin.Plugin{
	"state": &StateRPCPlugin{},
}

// StateRPCPlugin is the implementation of plugin.Plugin so we can serve/consume this.
type StateRPCPlugin struct {
	// Impl Injection - this will be provided by the plugin binary
	Impl StatePlugin
}

func (p *StateRPCPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &StateRPCServer{Impl: p.Impl}, nil
}

func (p *StateRPCPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &StateRPCClient{client: c}, nil
}

// StateRPCClient is an implementation that talks over RPC
type StateRPCClient struct {
	client *rpc.Client
}

func (c *StateRPCClient) Change(action string, data map[string]interface{}) error {
	var resp error
	args := &ChangeArgs{
		Action: action,
		Data:   data,
	}
	err := c.client.Call("Plugin.Change", args, &resp)
	if err != nil {
		return err
	}
	return resp
}

func (c *StateRPCClient) GetState() (json.RawMessage, error) {
	var resp json.RawMessage
	err := c.client.Call("Plugin.GetState", new(interface{}), &resp)
	return resp, err
}

// StateRPCServer is the RPC server that StateRPCClient talks to
type StateRPCServer struct {
	Impl StatePlugin
}

func (s *StateRPCServer) Change(args *ChangeArgs, resp *error) error {
	*resp = s.Impl.Change(args.Action, args.Data)
	return nil
}

func (s *StateRPCServer) GetState(args interface{}, resp *json.RawMessage) error {
	state, err := s.Impl.GetState()
	*resp = state
	return err
}

// ChangeArgs represents the arguments for Change RPC call
type ChangeArgs struct {
	Action string
	Data   map[string]interface{}
}
