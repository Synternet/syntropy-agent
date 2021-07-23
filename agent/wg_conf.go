package agent

import "log"

type wgConfReq struct {
	messageHeader
	/*
			"{
		    ""id"": ""string"",
		    ""type"": ""WG_CONF"",
		    ""data"": [
		        {
		            ""fn"": ""create_interface"",
		            ""args"": {
		                ""ifname"": ""interface_name"",
		                ""internal_ip"": ""10.5.9.12/29"",
		                ""listen_port?"": 49934
		            },
		            ""metadata"": {
		                ""network_id "": 1
		            }
		        },
		        {
		            ""fn"": ""add_peer"",
		            ""args"": {
		                ""ifname"": ""interface_name"",
		                ""public_key"": ""arn/vcAPOKJa+20ox1kIrDzUF/0eU7uYlw9wGe7pvTM="",
		                ""allowed_ips"": [
		                    ""8.8.8.8/32"",
		                    ""8.8.4.4/32""
		                ],
		                ""gw_ipv4"": ""10.5.9.12"",
		                ""endpoint_ipv4"": ""169.50.184.200"",  //optional
		                ""endpoint_port"": 49934 //optional
		            },
		            ""metadata"": {
		                ""device_id"": ""agent_dev_id"",
		                ""device_name"": ""device_name"",
		                ""device_public_ipv4"": ""72.5.6.4"",
		                ""connection_id"": 2
		                allowed_ips_info: [
		                    {
		                        agent_service_name: ""test_postgres333"",
		                        agent_service_tcp_ports: [5432, 5437],
		                        agent_service_udp_ports: [],
		                        agent_service_subnet_ip: ""172.19.0.5"",
		                    },
		                    {
		                        agent_service_name: ""test_postgres"",
		                        agent_service_tcp_ports: [5433, 5438],
		                        agent_service_udp_ports: [],
		                        agent_service_subnet_ip: ""172.0.0.3"",
		                    },
		                ],
		            }
		        },
		        {
		            ""fn"": ""remove_interface"",
		            ""args"": {
		                ""ifname"": ""interface_name""
		            }
		        },
		        {
		            ""fn"": ""remove_peer"",
		            ""args"": {
		                ""ifname"": ""interface_name"",
		                ""public_key"": ""arn/vcAPOKJa+20ox1kIrDzUF/0eU7uYlw9wGe7pvTM=""
		               ""allowed_ips"": ['10.0.23.21', '192.169.0.4']
		            }
		        }
		    ]
		}"
	*/
}

func wireguardConfigure(a *Agent, raw []byte) error {
	log.Println(string(raw))

	a.Transmit(raw)
	return nil
}
