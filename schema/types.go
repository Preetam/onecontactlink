package schema

type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	MainEmail int    `json:"mainEmail"`
	Created   int    `json:"created"`
	Updated   int    `json:"updated"`
	Deleted   int    `json:"deleted"`
}

type Email struct {
	ID      int    `json:"id"`
	Address string `json:"address"`
	User    int    `json:"user"`
	Created int    `json:"created"`
	Updated int    `json:"updated"`
	Deleted int    `json:"deleted"`
}

type Token struct {
	ID      int    `json:"id"`
	User    int    `json:"user"`
	Value   string `json:"value"`
	Created int    `json:"created"`
	Updated int    `json:"updated"`
	Deleted int    `json:"deleted"`
}

type Request struct {
	ID      int    `json:"id"`
	Code    string `json:"code"`
	From    int    `json:"from"`
	To      int    `json:"to"`
	Status  int    `json:"status"`
	Created int    `json:"created"`
	Updated int    `json:"updated"`
	Deleted int    `json:"deleted"`
}
