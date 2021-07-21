package agent

// Generic message struct (common part for all messages)
type messageHeader struct {
	ID      string `json:"id"`
	MsgType string `json:"type"`
}

type errorResponce struct {
	messageHeader
	Data struct {
		Type    string `json:"type"`
		Message string `json:"error"`
	} `json:"data"`
}

type autoPingRequest struct {
	messageHeader
	Data struct {
		IPs       []string `json:"ips"`
		Interval  int      `json:"interval"`
		RespLimit int      `json:"responce_limit"`
	} `json:"data"`
}

type autoPingResponce struct {
	messageHeader
	Data struct {
		Pings []struct {
			IP      string  `json:"ip"`
			Latency int     `json:"latency_ms"`
			Loss    float32 `json:"packet_loss"`
		} `json:"pings"`
	} `json:"data"`
}

type getInfoRequest struct {
	messageHeader
	Data interface{} `json:"data,omitempty"`
}

type NetworkInfoEntry struct {
	ID      string   `json:"agent_network_id,omitempty"`
	Name    string   `json:"agent_network_name,omitempty"`
	Subnets []string `json:"agent_network_subnets,omitempty"`
	// TODO: review why different names in array ?
	/*
		""network_info?"": [
			{
				""agent_network_id"": ""agent_network_id"",
				""agent_network_name"": ""agent_network_name"",
				""agent_network_subnets"": [
					""1.2.3.4/12""
				]
			},
			{
				""docker_network_id"": ""agent_network_id"",
				""docker_network_name"": ""agent_network_name"",
				""docker_network_subnets"": [
					""1.2.3.4/12""
				]
			}
		]
	*/
}

type ContainerInfoEntry struct {
	ID   string `json:"agent_container_id,omitempty"`
	Name string `json:"agent_container_name,omitempty"`

	// TODO: review for complete structure
	/*
		""container_info"":  [
				{
					""agent_container_id"": ""4e67bdb06bb2a9e19a61ad5a420b8701115263fe56b2918547cc9138084bf1c9"",
					""agent_container_name"": ""pgadmin"",
					""agent_container_networks: [aaa,bbb],
					""agent_container_ips"": [""172.18.0.2""],
					""agent_container_ports"": {""udp"": [], ""tcp"": [443, 5050, 5050, 80]},
					""agent_container_state"": ""running"",
					""agent_container_uptime"": ""Up About a minute""
				},
				{
					""agent_container_id"": ""5d1774cb76c9385dcd025abbc84faea12dc7d7f247597042882361a7baa86fe6"",
					""agent_container_name"": ""postgres"",
					""agent_container_ips: ['aaa', 'bbb'],
					""agent_container_subnets"": [""172.18.0.3/16""],
					""agent_container_ports"": {
							""udp"": [],
							""tcp"": [5432, 5435]
					},
					""agent_container_state"": ""running"",
					""agent_container_uptime"": ""Up About a minute""
				}
		]
	*/
}

type getInfoResponce struct {
	messageHeader
	Data struct {
		Provider   int      `json:"agent_provider,omitempty"` // 0 is not used and do not send
		Status     bool     `json:"service_status"`
		Tags       []string `json:"agent_tags"`
		ExternalIP string   `json:"external_ip"`

		NetworkInfo   []NetworkInfoEntry   `json:"network_info"`
		ContainerInfo []ContainerInfoEntry `json:"container_info"`
	} `json:"data"`
}

type configInfo struct {
	messageHeader
	/*
		"{
			""id"": ""string"",
			""type:"": ""CONFIG_INFO"",
			""data"": {
				""agent_id"": number,
				""network"": {
					 ""PUBLIC"": { internal_ip, public_key, listen_port },
					 ""SDN1"": ""{ internal_ip, public_key, listen_port } ,
					 ""SDN2"": { internal_ip, public_key, listen_port } ,
					 ""SDN3"": { internal_ip, public_key, listen_port } ,
				}
				""vpn"": [
					{
						""fn"": ""create_interface"",
						""args"": {
							""ifname"": ""p2p_0zlwc4iqbu"",
							""internal_ip"": ""10.69.0.10/31""
						},
					   ""metadata"": {
							""network_id"":1
					   }
					},
					{
						""fn"": ""add_peer"",
						""args"": {
							""allowed_ips"": [
								""10.69.0.10/32"",
								""150.150.150.150/32""
							],
							""endpoint_ipv4"": ""75.75.75.72"",
							""endpoint_port"": 49934,
							""ifname"": ""p2p_0zlwc4iqbu"",
							""public_key"": ""some-public-key"",
							""gw_ipv4"": ""10.69.0.10""
						},
						""metadata"": {
							""device_id"": ""agent_dev_id"",
							""device_name"": ""device_name"",
							""device_public_ipv4"": ""72.5.6.4"",
							""connection_id"":1
						}
					},
					{
						""fn"": ""create_interface"",
						""args"": {
							""ifname"": ""p2p_rgt2f3cten"",
							""internal_ip"": ""10.69.0.11/31""
						},
						""metadata"": {
							""network_id"":1
					   }
					},
					{
						""fn"": ""add_peer"",
						""args"": {
							""allowed_ips"": [
								""10.69.0.11/32"",
								""150.150.150.150/32""
							],
							""endpoint_ipv4"": ""75.75.75.68"",
							""endpoint_port"": 49934,
							""ifname"": ""p2p_rgt2f3cten"",
							""public_key"": ""some-public-key"",
							""gw_ipv4"": ""10.69.0.11""
						},
						""metadata"": {
							""device_id"": ""agent_dev_id"",
							""device_name"": ""device_name"",
							""device_public_ipv4"": ""72.5.6.4"",
							""connection_id"":1
						}
					}
				]
			}
		}"
	*/
}
