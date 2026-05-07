package core

type Comic struct {
	ID  int
	URL string
}

type AdminStats struct {
	WordsTotal    int
	WordsUnique   int
	ComicsFetched int
	ComicsTotal   int
}

type JobStatus string
