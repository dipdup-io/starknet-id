package postgres

import (
	"context"
	"database/sql"
	"encoding/hex"
	"testing"
	"time"

	"github.com/dipdup-io/starknet-id/internal/storage"
	"github.com/dipdup-net/go-lib/config"
	"github.com/dipdup-net/go-lib/database"
	"github.com/go-testfixtures/testfixtures/v3"
	"github.com/stretchr/testify/suite"
)

// StorageTestSuite -
type StorageTestSuite struct {
	suite.Suite
	psqlContainer *database.PostgreSQLContainer
	storage       Storage
}

// SetupSuite -
func (s *StorageTestSuite) SetupSuite() {
	ctx, ctxCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer ctxCancel()

	psqlContainer, err := database.NewPostgreSQLContainer(ctx, database.PostgreSQLContainerConfig{
		User:     "user",
		Password: "password",
		Database: "db_test",
		Port:     5432,
		Image:    "postgres:15",
	})
	s.Require().NoError(err)
	s.psqlContainer = psqlContainer

	storage, err := Create(ctx, config.Database{
		Kind:     config.DBKindPostgres,
		User:     s.psqlContainer.Config.User,
		Database: s.psqlContainer.Config.Database,
		Password: s.psqlContainer.Config.Password,
		Host:     s.psqlContainer.Config.Host,
		Port:     s.psqlContainer.MappedPort().Int(),
	})
	s.Require().NoError(err)
	s.storage = storage

	db, err := sql.Open("postgres", s.psqlContainer.GetDSN())
	s.Require().NoError(err)

	fixtures, err := testfixtures.New(
		testfixtures.Database(db),
		testfixtures.Dialect("postgres"),
		testfixtures.Directory("fixtures"),
	)
	s.Require().NoError(err)
	s.Require().NoError(fixtures.Load())
	s.Require().NoError(db.Close())
}

// TearDownSuite -
func (s *StorageTestSuite) TearDownSuite() {
	ctx, ctxCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer ctxCancel()

	s.Require().NoError(s.storage.Close())
	s.Require().NoError(s.psqlContainer.Terminate(ctx))
}

func (s *StorageTestSuite) TestGetByName() {
	ctx, ctxCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer ctxCancel()

	state, err := s.storage.State.ByName(ctx, "test")
	s.Require().NoError(err)
	s.Require().EqualValues(1, state.ID)
}

func (s *StorageTestSuite) TestGetByNameFailed() {
	ctx, ctxCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer ctxCancel()

	_, err := s.storage.State.ByName(ctx, "unknown")
	s.Require().Error(err)
}

func (s *StorageTestSuite) TestGetByHash() {
	ctx, ctxCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer ctxCancel()

	b, err := hex.DecodeString("0327d34747122d7a40f4670265b098757270a449ec80c4871450fffdab7c2fa8")
	s.Require().NoError(err)

	address, err := s.storage.Addresses.GetByHash(ctx, b)
	s.Require().NoError(err)
	s.Require().EqualValues(16, address.Id)
	s.Require().EqualValues(1, address.Height)
	s.Require().NotNil(address.ClassId)
	s.Require().EqualValues(1, *address.ClassId)
}

func (s *StorageTestSuite) TestGetByResolverId() {
	ctx, ctxCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer ctxCancel()

	subdomain, err := s.storage.Subdomains.GetByResolverId(ctx, 1673800)
	s.Require().NoError(err)
	s.Require().EqualValues(2, subdomain.Id)
	s.Require().EqualValues(1673800, subdomain.ResolverId)
	s.Require().EqualValues(50359, subdomain.RegistrationHeight)
	s.Require().Equal("xplorer", subdomain.Subdomain)

	_, err = s.storage.Subdomains.GetByResolverId(ctx, 1)
	s.Require().Error(err)
}

func (s *StorageTestSuite) TestTxSaveState() {
	ctx, ctxCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer ctxCancel()

	tx, err := BeginTransaction(ctx, s.storage.Transactable)
	s.Require().NoError(err)
	defer tx.Close(ctx)

	err = tx.SaveState(ctx, &storage.State{
		Name:       "test",
		LastHeight: 101,
		LastTime:   time.Now(),
	})
	s.Require().NoError(err)

	err = tx.Flush(ctx)
	s.Require().NoError(err)

	response, err := s.storage.State.ByName(ctx, "test")
	s.Require().NoError(err)
	s.Require().EqualValues(1, response.ID)
	s.Require().EqualValues("test", response.Name)
	s.Require().EqualValues(101, response.LastHeight)
}

func TestSuiteStorage_Run(t *testing.T) {
	suite.Run(t, new(StorageTestSuite))
}
