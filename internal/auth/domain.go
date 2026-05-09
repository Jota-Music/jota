package auth

type User struct {
	Name string `json:"name" xml:"name" form:"name" msgpack:"name"`
	Pass string `json:"pass" xml:"pass" form:"pass" msgpack:"pass"`
}
