package core

type Comics struct {
	ID  int
	URL string
}

type ComicsWithWords struct {
	ID    int
	URL   string
	Words []string
}
