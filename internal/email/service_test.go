package email

import (
	"context"
	"github.com/stretchr/testify/suite"
	"testing"
)

type dataMapperMock struct {
	pool []Email
}

func (m *dataMapperMock) FindReady(_ context.Context) ([]Email, error) {
	return m.pool, nil
}

type lockDataMapperMock struct {
	locks []Lock
}

func (m *lockDataMapperMock) BatchInsert(_ context.Context, locks []Lock) ([]Lock, error) {
	m.locks = locks
	return locks, nil
}

func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, &ServiceTestSuite{})
}

type ServiceTestSuite struct {
	suite.Suite
	dataMapper     *dataMapperMock
	lockDataMapper *lockDataMapperMock
}

func (suite *ServiceTestSuite) SetupTest() {
	suite.dataMapper = &dataMapperMock{}
	suite.lockDataMapper = &lockDataMapperMock{}
}

func (suite *ServiceTestSuite) Test_Finder_FindAndLock() {
	ready := Email{Id: "fake-id"}
	suite.dataMapper.pool = []Email{ready}

	sut := &Service{emailDataMapper: suite.dataMapper, lockDataMapper: suite.lockDataMapper}
	found, err := sut.LockAndReturnReadyToProcess(context.TODO())

	suite.Require().Nil(err)
	suite.Require().Equal(1, len(found))
	suite.Assert().Equal(ready.Id, found[0].Id)
	suite.Require().Equal(1, len(suite.lockDataMapper.locks))
	suite.Assert().Equal(ready.Id, suite.lockDataMapper.locks[0].Id)
}
