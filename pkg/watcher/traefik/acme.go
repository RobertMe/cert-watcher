package traefik

type acmeProvider struct {
	Certificates []struct {
		Domain struct {
			Main string `json:"main"`
			Sans []string `json:"sans"`
		} `json:"domain"`
		Certificate string `json:"certificate"`
		Key string `json:"key"`
	} `json:"Certificates"`
}
