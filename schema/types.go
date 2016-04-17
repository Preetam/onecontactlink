package schema

const (
	UserCodeSize    = 6
	RequestCodeSize = 32
	TokenSize       = 32
	LinkCodeSize    = 32
)

type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	MainEmail string `json:"mainEmail"`
	Status    int    `json:"status"`
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
	ID       int    `json:"id"`
	Code     string `json:"code"`
	FromUser int    `json:"fromUser"`
	ToUser   int    `json:"toUser"`
	Status   int    `json:"status"`
	Created  int    `json:"created"`
	Updated  int    `json:"updated"`
	Deleted  int    `json:"deleted"`
}

type RequestLink struct {
	ID      int    `json:"id"`
	User    int    `json:"user"`
	Code    string `json:"code"`
	Created int    `json:"created"`
	Updated int    `json:"updated"`
	Deleted int    `json:"deleted"`
}

func NewUser(name, email string) *User {
	return &User{
		Name:      name,
		MainEmail: email,
	}
}
