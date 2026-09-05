package auth_controller

import "golang.org/x/oauth2"

func oauth2AuthURLOptions() []oauth2.AuthCodeOption {
	return []oauth2.AuthCodeOption{
		oauth2.AccessTypeOnline,
		oauth2.SetAuthURLParam("show_dialog", "true"),
	}
}
