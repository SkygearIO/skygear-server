package userprofile

type Store interface {
	CreateUserProfile(userID string, data Data) (UserProfile, error)
	GetUserProfile(userID string) (UserProfile, error)
	UpdateUserProfile(userID string, data Data) (UserProfile, error)
}
