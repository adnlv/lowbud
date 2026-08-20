package domain

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
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

type GetBalanceQuery struct {
	AccountID string
}

type GetBalanceResult struct {
	Amount decimal.Decimal `json:"amount"`
}

type GetTransactionQuery struct {
	AccountID     string
	TransactionID string
}

type GetTransactionResult struct {
	ID          string
	Description string
	CreatedAt   time.Time
	Amount      decimal.Decimal
}

type GetTransactionHistoryQuery struct {
	Page      uint64
	PerPage   uint64
	AccountID string
}

type TransactionHistoryItem struct {
	ID          string
	Description string
	CreatedAt   time.Time
	Amount      decimal.Decimal
}

type TransferFundsCommand struct {
	IdempotencyKey       string
	Description          string
	SourceAccountID      string
	DestinationAccountID string
	Amount               decimal.Decimal
}

type LedgerService interface {
	GetBalance(ctx context.Context, getBalanceQuery *GetBalanceQuery) (*GetBalanceResult, error)
	GetTransaction(ctx context.Context, getTransactionQuery *GetTransactionQuery) (*GetTransactionResult, error)
	GetTransactionHistory(ctx context.Context, transactionHistoryQuery *GetTransactionHistoryQuery) ([]TransactionHistoryItem, error)
	TransferFunds(ctx context.Context, transferFundsCommand *TransferFundsCommand) error
}

type ledgerServiceImpl struct {
	DB           *pgxpool.Pool
	UUIDProvider UUIDProvider
}

func NewLedgerService(db *pgxpool.Pool, uuidProvider UUIDProvider) LedgerService {
	return &ledgerServiceImpl{
		DB:           db,
		UUIDProvider: uuidProvider,
	}
}

func (s *ledgerServiceImpl) GetBalance(ctx context.Context, getBalanceQuery *GetBalanceQuery) (*GetBalanceResult, error) {
	const q = `
SELECT COALESCE(SUM(l.amount), 0)
FROM accounts a
LEFT JOIN ledger_entries l ON l.account_id = a.id
WHERE a.id = $1 AND a.closed_at IS NULL
GROUP BY a.id;
`
	var balance decimal.Decimal
	if err := s.DB.QueryRow(ctx, q, getBalanceQuery.AccountID).Scan(&balance); err != nil {
		return nil, err
	}
	return &GetBalanceResult{Amount: balance}, nil
}

func (s *ledgerServiceImpl) GetTransaction(ctx context.Context, getTransactionQuery *GetTransactionQuery) (*GetTransactionResult, error) {
	if err := s.UUIDProvider.ValidateUUID(getTransactionQuery.TransactionID); err != nil {
		return nil, err
	}

	const q = `
SELECT 
    t.id,
    t.description,
    t.created_at,
    e.amount
FROM ledger_transactions t
JOIN ledger_entries e ON e.ledger_transaction_id = t.id
WHERE t.id = $1 AND e.account_id = $2;
`
	var transaction LedgerTransaction
	var entry LedgerEntry
	if err := s.DB.QueryRow(
		ctx,
		q,
		getTransactionQuery.TransactionID,
		getTransactionQuery.AccountID,
	).Scan(
		&transaction.ID,
		&transaction.Description,
		&transaction.CreatedAt,
		&entry.Amount,
	); err != nil {
		return nil, err
	}
	return &GetTransactionResult{
		ID:          transaction.ID,
		Description: transaction.Description,
		CreatedAt:   transaction.CreatedAt,
		Amount:      entry.Amount,
	}, nil
}

func (s *ledgerServiceImpl) GetTransactionHistory(ctx context.Context, transactionHistoryQuery *GetTransactionHistoryQuery) ([]TransactionHistoryItem, error) {
	const q = `
SELECT 
    t.id,
    t.description,
    t.created_at,
    e.amount
FROM ledger_entries e
JOIN ledger_transactions t ON t.id = e.ledger_transaction_id
WHERE e.account_id = $1
ORDER BY e.created_at DESC
LIMIT $2
OFFSET $3
`
	offset := (transactionHistoryQuery.Page - 1) * transactionHistoryQuery.PerPage
	rows, err := s.DB.Query(ctx, q, transactionHistoryQuery.AccountID, transactionHistoryQuery.PerPage, offset)
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[TransactionHistoryItem])
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (s *ledgerServiceImpl) TransferFunds(ctx context.Context, transferFundsCommand *TransferFundsCommand) error {
	if !transferFundsCommand.Amount.IsPositive() {
		return errors.New("negative and zero transfers are rejected")
	}

	if transferFundsCommand.SourceAccountID == transferFundsCommand.DestinationAccountID {
		return errors.New("can't transfer funds to themself")
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const lockSenderQuery = `
SELECT id FROM accounts WHERE id = $1 AND closed_at IS NULL FOR UPDATE
`
	var senderId string
	if err = tx.QueryRow(ctx, lockSenderQuery, transferFundsCommand.SourceAccountID).Scan(&senderId); err != nil {
		return errors.New("sender account not found or closed")
	}

	const checkDestinationQuery = `
SELECT id FROM accounts WHERE id = $1 AND closed_at IS NULL
`
	var destinationId string
	if err = tx.QueryRow(ctx, checkDestinationQuery, transferFundsCommand.DestinationAccountID).Scan(&destinationId); err != nil {
		return errors.New("destination account not found or closed")
	}

	const totalBalanceQuery = `
SELECT COALESCE(SUM(amount), 0) FROM ledger_entries WHERE account_id = $1
`
	var balance decimal.Decimal
	if err = tx.QueryRow(ctx, totalBalanceQuery, transferFundsCommand.SourceAccountID).Scan(&balance); err != nil {
		return err
	}

	if transferFundsCommand.Amount.GreaterThan(balance) {
		return errors.New("not enough money")
	}

	transactionId, err := s.UUIDProvider.NewUUID()
	if err != nil {
		return err
	}

	transaction := &LedgerTransaction{
		ID:             transactionId,
		IdempotencyKey: transferFundsCommand.IdempotencyKey,
		Description:    transferFundsCommand.Description,
		CreatedAt:      time.Now(),
	}

	const insertTransactionQuery = `
INSERT INTO ledger_transactions (id, idempotency_key, description, created_at) 
VALUES ($1, $2, $3, $4)
`
	if _, err = tx.Exec(
		ctx,
		insertTransactionQuery,
		transaction.ID,
		transaction.IdempotencyKey,
		transaction.Description,
		transaction.CreatedAt,
	); err != nil {
		return err
	}

	creditEntryId, err := s.UUIDProvider.NewUUID()
	if err != nil {
		return err
	}

	creditEntry := &LedgerEntry{
		ID:                  creditEntryId,
		LedgerTransactionID: transactionId,
		AccountID:           transferFundsCommand.SourceAccountID,
		Amount:              transferFundsCommand.Amount.Neg(),
		CreatedAt:           time.Now(),
	}

	debitEntryId, err := s.UUIDProvider.NewUUID()
	if err != nil {
		return err
	}

	debitEntry := &LedgerEntry{
		ID:                  debitEntryId,
		LedgerTransactionID: transactionId,
		AccountID:           transferFundsCommand.DestinationAccountID,
		Amount:              transferFundsCommand.Amount,
		CreatedAt:           time.Now(),
	}

	const insertLedgerEntryQuery = `
INSERT INTO ledger_entries (id, ledger_transaction_id, account_id, amount, created_at) 
VALUES ($1, $2, $3, $4, $5)
`
	if _, err = tx.Exec(
		ctx,
		insertLedgerEntryQuery,
		creditEntry.ID,
		creditEntry.LedgerTransactionID,
		creditEntry.AccountID,
		creditEntry.Amount,
		creditEntry.CreatedAt,
	); err != nil {
		return err
	}

	if _, err = tx.Exec(
		ctx,
		insertLedgerEntryQuery,
		debitEntry.ID,
		debitEntry.LedgerTransactionID,
		debitEntry.AccountID,
		debitEntry.Amount,
		debitEntry.CreatedAt,
	); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}
