package main

type ProductVersion struct {
	Branch    string `json:"branch"`
	HostDocs  bool   `json:"hostDocs"`
	VDropdown bool   `json:"v-dropdown,omitempty"`
	DocsDir   string   `json:"docsDir,omitempty"` // default: "docs"
}

type Product struct {
	Name string `json:"-"`
	URL           string           `json:"url"`
	Versions      []ProductVersion `json:"versions"`
	LatestVersion string           `json:"latestVersion"`
	GithubURL     string           `json:"githubUrl"`
}

type DocAggregator struct {
	Products map[string]Product `json:"products"`
}
