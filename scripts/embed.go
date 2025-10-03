package scripts

import _ "embed"

// NetworkAgentScript contains the node-side probe script shipped with the controller.
//
//go:embed node-network-agent.sh
var NetworkAgentScript string
