package entities

func (c AgentOrchestrationItemConfig) GetProviderType() (string, string) {
	return "agent_orchestration", "app"
}

type AgentOrchestrationItemConfig struct {
	Version          string `json:"version"`
	AgentPool        string `json:"agent_pool"`
	ChannelURL       string `json:"channel_url"`
	NetSegment       string `json:"net_segment"`
	AgentInstance    string `json:"agent_instance"`
	CountOfExecutors int64  `json:"count_of_executors"`
}

type JenkinsAgentSubsystem struct {
	RisID         JenkinsAgentSubsystemValue `json:"ris_id"`
	IsCode        JenkinsAgentSubsystemValue `json:"is_code"`
	HeadName      JenkinsAgentSubsystemValue `json:"head_name"`
	NetSegment    JenkinsAgentSubsystemValue `json:"net_segment"`
	DisplayName   JenkinsAgentSubsystemValue `json:"display_name"`
	SferaHeadURL  JenkinsAgentSubsystemValue `json:"sfera_head_url"`
	NodeGroupName JenkinsAgentSubsystemValue `json:"node_group_name"`
}

type JenkinsAgentSubsystemValue struct {
	Value string `json:"value"`
}
