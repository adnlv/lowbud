package domain

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RegisterAccountCommand struct {
	Email    string
	Forename string
	Surname  string
	Password string
}

type BasicLoginCommand struct {
	Email    string
	Password string
}

type BasicLoginResult struct {
	AccessToken string
}

type AuthService interface {
	BasicLogin(ctx context.Context, basicLoginCommand *BasicLoginCommand) (*BasicLoginResult, error)
	RegisterAccount(ctx context.Context, registerAccountCommand *RegisterAccountCommand) error
	ParseAccessToken(ctx context.Context, tokenStr string) (*AccessTokenClaims, error)
}

type authServiceImpl struct {
	DB                  *pgxpool.Pool
	UUIDProvider        UUIDProvider
	PasswordHasher      PasswordHasher
	AccessTokenProvider AccessTokenProvider
}

func NewAuthService(
	db *pgxpool.Pool,
	uuidProvider UUIDProvider,
	passwordHasher PasswordHasher,
	accessTokenProvider AccessTokenProvider,
) AuthService {
	return &authServiceImpl{
		DB:                  db,
		UUIDProvider:        uuidProvider,
		PasswordHasher:      passwordHasher,
		AccessTokenProvider: accessTokenProvider,
	}
}

func (s *authServiceImpl) BasicLogin(ctx context.Context, basicLoginCommand *BasicLoginCommand) (*BasicLoginResult, error) {
	account := &Account{Email: basicLoginCommand.Email}

	const selectAccountByIdQuery = `
SELECT id, password_hash FROM accounts WHERE email = $1 AND closed_at IS NULL
`
	if err := s.DB.QueryRow(
		ctx,
		selectAccountByIdQuery,
		account.Email,
	).Scan(
		&account.ID,
		&account.PasswordHash,
	); err != nil {
		return nil, err
	}

	if err := s.PasswordHasher.CompareHashAndPassword(account.PasswordHash, basicLoginCommand.Password); err != nil {
		return nil, err
	}

	accessToken, err := s.AccessTokenProvider.NewAccessToken(&AccessTokenClaims{AccountID: account.ID})
	if err != nil {
		return nil, err
	}

	return &BasicLoginResult{AccessToken: accessToken}, nil
}

func (s *authServiceImpl) RegisterAccount(ctx context.Context, registerAccountCommand *RegisterAccountCommand) error {
	accountId, err := s.UUIDProvider.NewUUID()
	if err != nil {
		return err
	}

	passwordHash, err := s.PasswordHasher.HashPassword(registerAccountCommand.Password)
	if err != nil {
		return err
	}

	account := &Account{
		ID:           accountId,
		Forename:     registerAccountCommand.Forename,
		Surname:      registerAccountCommand.Surname,
		Email:        registerAccountCommand.Email,
		PasswordHash: passwordHash,
		RegisteredAt: time.Now(),
		UpdatedAt:    time.Now(),
	}

	const insertAccountQuery = `
INSERT INTO accounts (id, forename, surname, email, password_hash, registered_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
`
	if _, err = s.DB.Exec(
		ctx,
		insertAccountQuery,
		account.ID,
		account.Forename,
		account.Surname,
		account.Email,
		account.PasswordHash,
		account.RegisteredAt,
		account.UpdatedAt,
	); err != nil {
		return err
	}

	return nil
}

func (s *authServiceImpl) ParseAccessToken(_ context.Context, tokenStr string) (*AccessTokenClaims, error) {
	claims, err := s.AccessTokenProvider.ParseAccessToken(tokenStr)
	if err != nil {
		return nil, err
	}
	return claims, nil
}

type GetAccountInfoQuery struct {
	AccountID string
}

type GetAccountInfoResult struct {
	Email        string
	Forename     string
	Surname      string
	RegisteredAt time.Time
}

type CloseAccountCommand struct {
	AccountID string
}

type AccountService interface {
	GetAccountInfo(ctx context.Context, getAccountInfoQuery *GetAccountInfoQuery) (*GetAccountInfoResult, error)
	CloseAccount(ctx context.Context, closeAccountCommand *CloseAccountCommand) error
}

type accountServiceImpl struct {
	DB *pgxpool.Pool
}

func NewAccountService(db *pgxpool.Pool) AccountService {
	return &accountServiceImpl{DB: db}
}

func (s *accountServiceImpl) GetAccountInfo(ctx context.Context, getAccountInfoQuery *GetAccountInfoQuery) (*GetAccountInfoResult, error) {
	account := &Account{ID: getAccountInfoQuery.AccountID}

	const getAccountByIdQuery = `
SELECT email, forename, surname, registered_at FROM accounts WHERE id = $1 AND closed_at IS NULL
`
	if err := s.DB.QueryRow(
		ctx,
		getAccountByIdQuery,
		account.ID,
	).Scan(
		&account.Email,
		&account.Forename,
		&account.Surname,
		&account.RegisteredAt,
	); err != nil {
		return nil, err
	}

	return &GetAccountInfoResult{
		Email:        account.Email,
		Forename:     account.Forename,
		Surname:      account.Surname,
		RegisteredAt: account.RegisteredAt,
	}, nil
}

func (s *accountServiceImpl) CloseAccount(ctx context.Context, closeAccountCommand *CloseAccountCommand) error {
	account := &Account{
		ID:       closeAccountCommand.AccountID,
		ClosedAt: new(time.Now()),
	}

	const closeAccountByIdQuery = `
UPDATE accounts SET closed_at = $2 WHERE id = $1 AND closed_at IS NULL
`
	if _, err := s.DB.Exec(
		ctx,
		closeAccountByIdQuery,
		account.ID,
		account.ClosedAt,
	); err != nil {
		return err
	}

	return nil
}
