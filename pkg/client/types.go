package client

type CheckRun struct {
	ID         int64
	HeadSHA    string
	Status     string
	Conclusion string

	Suite *CheckSuite
}

type CheckSuite struct {
	ID         int64
	HeadSHA    string
	Status     string
	Conclusion string
}

type CheckEvent struct {
	Action     string
	IsCheckRun bool
	Run        *CheckRun

	IsCheckSuite bool
	Suite        *CheckSuite
}

type PullRequest struct {
	Number             int
	State              string
	Title              string
	Body               string
	Labels             []string
	User               string
	HTMLURL            string
	RequestedReviewers []string
}
