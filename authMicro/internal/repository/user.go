package repository

type User struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Surname  string `json:"surname"`
}

type IUserRepository interface {
	Save(user User) error
	FindByEmail(email string) (User, error)
}

type UserRepo struct {
	users map[string]User
}

func NewMemoryUserRepo() *UserRepo {
	return &UserRepo{
		users: make(map[string]User),
	}
}

func (m *UserRepo) Save(user User) error {
	m.users[user.Email] = user
	return nil
}

func (m *UserRepo) FindByEmail(email string) (User, error) {
	user, ok := m.users[email]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return user, nil
}
