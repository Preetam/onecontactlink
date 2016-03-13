package schema

type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	MainEmail int    `json:"mainEmail"`
	Created   int    `json:"created"`
	Updated   int    `json:"updated"`
	Deleted   int    `json:"deleted"`
}

type Token struct {
	ID      int    `json:"id"`
	User    int    `json:"user"`
	Value   string `json:"value"`
	Created int    `json:"created"`
	Updated int    `json:"updated"`
	Deleted int    `json:"deleted"`
}
