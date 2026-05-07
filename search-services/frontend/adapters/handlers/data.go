package handlers

import "github.com/1ncrease0/xkcd-search/frontend/core"

type searchData struct {
	Comics []core.Comic
}

type pageData struct {
	Search  *searchData
	Preview *previewData
	Admin   *adminData
	Error   string
}

type previewData struct {
	ID  int
	URL string
}

type adminData struct {
	Stats     *core.AdminStats
	Ping      map[string]string
	JobStatus core.JobStatus
}
