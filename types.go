package main

type ProductVersion struct {
	Branch    string `json:"branch"`
	HostDocs  bool   `json:"hostDocs"`
	VDropdown bool   `json:"v-dropdown,omitempty"`
}

type Product struct {
	URL           string           `json:"url"`
	Versions      []ProductVersion `json:"versions"`
	LatestVersion string           `json:"latestVersion"`
	GithubURL     string           `json:"githubUrl"`
}

type DocAggregator struct {
	Products map[string]Product `json:"products"`
}
