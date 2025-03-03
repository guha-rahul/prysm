package client

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	validatorType "github.com/prysmaticlabs/prysm/v5/consensus-types/validator"
	ethpb "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
	"github.com/prysmaticlabs/prysm/v5/testing/mock"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
	validatormock "github.com/prysmaticlabs/prysm/v5/testing/validator-mock"
	walletMock "github.com/prysmaticlabs/prysm/v5/validator/accounts/testing"
	"github.com/prysmaticlabs/prysm/v5/validator/client/iface"
	"github.com/prysmaticlabs/prysm/v5/validator/keymanager/derived"
	constant "github.com/prysmaticlabs/prysm/v5/validator/testing"
	logTest "github.com/sirupsen/logrus/hooks/test"
	mock2 "github.com/stretchr/testify/mock"
	"github.com/tyler-smith/go-bip39"
	util "github.com/wealdtech/go-eth2-util"
	"go.uber.org/mock/gomock"
)

func TestWaitActivation_ContextCanceled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	validatorClient := validatormock.NewMockValidatorClient(ctrl)
	chainClient := validatormock.NewMockChainClient(ctrl)
	kp := randKeypair(t)
	v := validator{
		validatorClient: validatorClient,
		km:              newMockKeymanager(t, kp),
		chainClient:     chainClient,
		pubkeyToStatus:  make(map[[48]byte]*validatorStatus),
	}
	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
	ctx, cancel := context.WithCancel(context.Background())
	validatorClient.EXPECT().WaitForActivation(
		gomock.Any(),
		&ethpb.ValidatorActivationRequest{
			PublicKeys: [][]byte{kp.pub[:]},
		},
	).Return(clientStream, nil)
	clientStream.EXPECT().Recv().Return(
		&ethpb.ValidatorActivationResponse{},
		nil,
	).Do(func() { cancel() })
	assert.ErrorContains(t, cancelledCtx, v.WaitForActivation(ctx, nil))
}

func TestWaitActivation_StreamSetupFails_AttemptsToReconnect(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	validatorClient := validatormock.NewMockValidatorClient(ctrl)
	chainClient := validatormock.NewMockChainClient(ctrl)
	prysmChainClient := validatormock.NewMockPrysmChainClient(ctrl)
	kp := randKeypair(t)
	v := validator{
		validatorClient:  validatorClient,
		km:               newMockKeymanager(t, kp),
		chainClient:      chainClient,
		prysmChainClient: prysmChainClient,
		pubkeyToStatus:   make(map[[48]byte]*validatorStatus),
	}
	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
	validatorClient.EXPECT().WaitForActivation(
		gomock.Any(),
		&ethpb.ValidatorActivationRequest{
			PublicKeys: [][]byte{kp.pub[:]},
		},
	).Return(clientStream, errors.New("failed stream")).Return(clientStream, nil)
	prysmChainClient.EXPECT().ValidatorCount(
		gomock.Any(),
		"head",
		[]validatorType.Status{validatorType.Active},
	).Return([]iface.ValidatorCount{}, nil)
	resp := generateMockStatusResponse([][]byte{kp.pub[:]})
	resp.Statuses[0].Status.Status = ethpb.ValidatorStatus_ACTIVE
	clientStream.EXPECT().Recv().Return(resp, nil)
	assert.NoError(t, v.WaitForActivation(context.Background(), nil))
}

func TestWaitForActivation_ReceiveErrorFromStream_AttemptsReconnection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	validatorClient := validatormock.NewMockValidatorClient(ctrl)
	chainClient := validatormock.NewMockChainClient(ctrl)
	prysmChainClient := validatormock.NewMockPrysmChainClient(ctrl)
	kp := randKeypair(t)
	v := validator{
		validatorClient:  validatorClient,
		km:               newMockKeymanager(t, kp),
		chainClient:      chainClient,
		prysmChainClient: prysmChainClient,
		pubkeyToStatus:   make(map[[48]byte]*validatorStatus),
	}
	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
	validatorClient.EXPECT().WaitForActivation(
		gomock.Any(),
		&ethpb.ValidatorActivationRequest{
			PublicKeys: [][]byte{kp.pub[:]},
		},
	).Return(clientStream, nil)
	prysmChainClient.EXPECT().ValidatorCount(
		gomock.Any(),
		"head",
		[]validatorType.Status{validatorType.Active},
	).Return([]iface.ValidatorCount{}, nil)
	// A stream fails the first time, but succeeds the second time.
	resp := generateMockStatusResponse([][]byte{kp.pub[:]})
	resp.Statuses[0].Status.Status = ethpb.ValidatorStatus_ACTIVE
	clientStream.EXPECT().Recv().Return(
		nil,
		errors.New("fails"),
	).Return(resp, nil)
	assert.NoError(t, v.WaitForActivation(context.Background(), nil))
}

func TestWaitActivation_LogsActivationEpochOK(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	validatorClient := validatormock.NewMockValidatorClient(ctrl)
	chainClient := validatormock.NewMockChainClient(ctrl)
	prysmChainClient := validatormock.NewMockPrysmChainClient(ctrl)
	kp := randKeypair(t)
	v := validator{
		validatorClient:  validatorClient,
		km:               newMockKeymanager(t, kp),
		genesisTime:      1,
		chainClient:      chainClient,
		prysmChainClient: prysmChainClient,
		pubkeyToStatus:   make(map[[48]byte]*validatorStatus),
	}
	resp := generateMockStatusResponse([][]byte{kp.pub[:]})
	resp.Statuses[0].Status.Status = ethpb.ValidatorStatus_ACTIVE
	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
	validatorClient.EXPECT().WaitForActivation(
		gomock.Any(),
		&ethpb.ValidatorActivationRequest{
			PublicKeys: [][]byte{kp.pub[:]},
		},
	).Return(clientStream, nil)
	prysmChainClient.EXPECT().ValidatorCount(
		gomock.Any(),
		"head",
		[]validatorType.Status{validatorType.Active},
	).Return([]iface.ValidatorCount{}, nil)
	clientStream.EXPECT().Recv().Return(
		resp,
		nil,
	)
	assert.NoError(t, v.WaitForActivation(context.Background(), nil), "Could not wait for activation")
	assert.LogsContain(t, hook, "Validator activated")
}

func TestWaitForActivation_Exiting(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	validatorClient := validatormock.NewMockValidatorClient(ctrl)
	chainClient := validatormock.NewMockChainClient(ctrl)
	prysmChainClient := validatormock.NewMockPrysmChainClient(ctrl)
	kp := randKeypair(t)
	v := validator{
		validatorClient:  validatorClient,
		km:               newMockKeymanager(t, kp),
		chainClient:      chainClient,
		prysmChainClient: prysmChainClient,
		pubkeyToStatus:   make(map[[48]byte]*validatorStatus),
	}
	resp := generateMockStatusResponse([][]byte{kp.pub[:]})
	resp.Statuses[0].Status.Status = ethpb.ValidatorStatus_EXITING
	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
	validatorClient.EXPECT().WaitForActivation(
		gomock.Any(),
		&ethpb.ValidatorActivationRequest{
			PublicKeys: [][]byte{kp.pub[:]},
		},
	).Return(clientStream, nil)
	prysmChainClient.EXPECT().ValidatorCount(
		gomock.Any(),
		"head",
		[]validatorType.Status{validatorType.Active},
	).Return([]iface.ValidatorCount{}, nil)
	clientStream.EXPECT().Recv().Return(
		resp,
		nil,
	)
	assert.NoError(t, v.WaitForActivation(context.Background(), nil))
}

func TestWaitForActivation_RefetchKeys(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.MainnetConfig().Copy()
	cfg.ConfigName = "test"
	cfg.SecondsPerSlot = 1
	params.OverrideBeaconConfig(cfg)
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	validatorClient := validatormock.NewMockValidatorClient(ctrl)
	chainClient := validatormock.NewMockChainClient(ctrl)
	prysmChainClient := validatormock.NewMockPrysmChainClient(ctrl)

	kp := randKeypair(t)
	km := newMockKeymanager(t)

	v := validator{
		validatorClient:  validatorClient,
		km:               km,
		chainClient:      chainClient,
		prysmChainClient: prysmChainClient,
		pubkeyToStatus:   make(map[[48]byte]*validatorStatus),
	}
	resp := generateMockStatusResponse([][]byte{kp.pub[:]})
	resp.Statuses[0].Status.Status = ethpb.ValidatorStatus_ACTIVE
	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
	validatorClient.EXPECT().WaitForActivation(
		gomock.Any(),
		&ethpb.ValidatorActivationRequest{
			PublicKeys: [][]byte{kp.pub[:]},
		},
	).Return(clientStream, nil)
	prysmChainClient.EXPECT().ValidatorCount(
		gomock.Any(),
		"head",
		[]validatorType.Status{validatorType.Active},
	).Return([]iface.ValidatorCount{}, nil)
	clientStream.EXPECT().Recv().Return(
		resp,
		nil)
	accountChan := make(chan [][fieldparams.BLSPubkeyLength]byte)
	sub := km.SubscribeAccountChanges(accountChan)
	defer func() {
		sub.Unsubscribe()
		close(accountChan)
	}()
	// update the accounts after a delay
	go func() {
		time.Sleep(2 * time.Second)
		require.NoError(t, km.add(kp))
		km.SimulateAccountChanges([][48]byte{kp.pub})
	}()
	assert.NoError(t, v.internalWaitForActivation(context.Background(), accountChan), "Could not wait for activation")
	assert.LogsContain(t, hook, msgNoKeysFetched)
	assert.LogsContain(t, hook, "Validator activated")
}

// Regression test for a scenario where you start with an inactive key and then import an active key.
func TestWaitForActivation_AccountsChanged(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("Imported keymanager", func(t *testing.T) {
		inactive := randKeypair(t)
		active := randKeypair(t)
		km := newMockKeymanager(t, inactive)
		validatorClient := validatormock.NewMockValidatorClient(ctrl)
		chainClient := validatormock.NewMockChainClient(ctrl)
		prysmChainClient := validatormock.NewMockPrysmChainClient(ctrl)
		v := validator{
			validatorClient:  validatorClient,
			km:               km,
			chainClient:      chainClient,
			prysmChainClient: prysmChainClient,
			pubkeyToStatus:   make(map[[48]byte]*validatorStatus),
		}
		inactiveResp := generateMockStatusResponse([][]byte{inactive.pub[:]})
		inactiveResp.Statuses[0].Status.Status = ethpb.ValidatorStatus_UNKNOWN_STATUS
		inactiveClientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
		validatorClient.EXPECT().WaitForActivation(
			gomock.Any(),
			&ethpb.ValidatorActivationRequest{
				PublicKeys: [][]byte{inactive.pub[:]},
			},
		).DoAndReturn(func(ctx context.Context, in *ethpb.ValidatorActivationRequest) (*mock.MockBeaconNodeValidator_WaitForActivationClient, error) {
			//delay a bit so that other key can be added
			time.Sleep(time.Second * 2)
			return inactiveClientStream, nil
		})
		prysmChainClient.EXPECT().ValidatorCount(
			gomock.Any(),
			"head",
			[]validatorType.Status{validatorType.Active},
		).Return([]iface.ValidatorCount{}, nil).AnyTimes()
		inactiveClientStream.EXPECT().Recv().Return(
			inactiveResp,
			nil,
		).AnyTimes()

		activeResp := generateMockStatusResponse([][]byte{inactive.pub[:], active.pub[:]})
		activeResp.Statuses[0].Status.Status = ethpb.ValidatorStatus_UNKNOWN_STATUS
		activeResp.Statuses[1].Status.Status = ethpb.ValidatorStatus_ACTIVE
		activeClientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
		validatorClient.EXPECT().WaitForActivation(
			gomock.Any(),
			mock2.MatchedBy(func(req *ethpb.ValidatorActivationRequest) bool {
				found := 0
				for _, pk := range req.PublicKeys {
					if bytes.Equal(pk, active.pub[:]) || bytes.Equal(pk, inactive.pub[:]) {
						found++
					}
				}
				return found == 2
			}),
		).Return(activeClientStream, nil)
		activeClientStream.EXPECT().Recv().Return(
			activeResp,
			nil,
		)

		go func() {
			// We add the active key into the keymanager and simulate a key refresh.
			time.Sleep(time.Second * 1)
			require.NoError(t, km.add(active))
			km.SimulateAccountChanges(make([][fieldparams.BLSPubkeyLength]byte, 0))
		}()

		assert.NoError(t, v.WaitForActivation(context.Background(), nil))
		assert.LogsContain(t, hook, "Waiting for deposit to be observed by beacon node")
		assert.LogsContain(t, hook, "Validator activated")
	})

	t.Run("Derived keymanager", func(t *testing.T) {
		seed := bip39.NewSeed(constant.TestMnemonic, "")
		inactivePrivKey, err :=
			util.PrivateKeyFromSeedAndPath(seed, fmt.Sprintf(derived.ValidatingKeyDerivationPathTemplate, 0))
		require.NoError(t, err)
		var inactivePubKey [fieldparams.BLSPubkeyLength]byte
		copy(inactivePubKey[:], inactivePrivKey.PublicKey().Marshal())
		activePrivKey, err :=
			util.PrivateKeyFromSeedAndPath(seed, fmt.Sprintf(derived.ValidatingKeyDerivationPathTemplate, 1))
		require.NoError(t, err)
		var activePubKey [fieldparams.BLSPubkeyLength]byte
		copy(activePubKey[:], activePrivKey.PublicKey().Marshal())
		wallet := &walletMock.Wallet{
			Files:            make(map[string]map[string][]byte),
			AccountPasswords: make(map[string]string),
			WalletPassword:   "secretPassw0rd$1999",
		}
		ctx := context.Background()
		km, err := derived.NewKeymanager(ctx, &derived.SetupConfig{
			Wallet:           wallet,
			ListenForChanges: true,
		})
		require.NoError(t, err)
		err = km.RecoverAccountsFromMnemonic(ctx, constant.TestMnemonic, derived.DefaultMnemonicLanguage, "", 1)
		require.NoError(t, err)
		validatorClient := validatormock.NewMockValidatorClient(ctrl)
		chainClient := validatormock.NewMockChainClient(ctrl)
		prysmChainClient := validatormock.NewMockPrysmChainClient(ctrl)
		v := validator{
			validatorClient:  validatorClient,
			km:               km,
			genesisTime:      1,
			chainClient:      chainClient,
			prysmChainClient: prysmChainClient,
			pubkeyToStatus:   make(map[[48]byte]*validatorStatus),
		}

		inactiveResp := generateMockStatusResponse([][]byte{inactivePubKey[:]})
		inactiveResp.Statuses[0].Status.Status = ethpb.ValidatorStatus_UNKNOWN_STATUS
		inactiveClientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
		validatorClient.EXPECT().WaitForActivation(
			gomock.Any(),
			&ethpb.ValidatorActivationRequest{
				PublicKeys: [][]byte{inactivePubKey[:]},
			},
		).DoAndReturn(func(ctx context.Context, in *ethpb.ValidatorActivationRequest) (*mock.MockBeaconNodeValidator_WaitForActivationClient, error) {
			//delay a bit so that other key can be added
			time.Sleep(time.Second * 2)
			return inactiveClientStream, nil
		})
		prysmChainClient.EXPECT().ValidatorCount(
			gomock.Any(),
			"head",
			[]validatorType.Status{validatorType.Active},
		).Return([]iface.ValidatorCount{}, nil).AnyTimes()
		inactiveClientStream.EXPECT().Recv().Return(
			inactiveResp,
			nil,
		).AnyTimes()

		activeResp := generateMockStatusResponse([][]byte{inactivePubKey[:], activePubKey[:]})
		activeResp.Statuses[0].Status.Status = ethpb.ValidatorStatus_UNKNOWN_STATUS
		activeResp.Statuses[1].Status.Status = ethpb.ValidatorStatus_ACTIVE
		activeClientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
		validatorClient.EXPECT().WaitForActivation(
			gomock.Any(),
			&ethpb.ValidatorActivationRequest{
				PublicKeys: [][]byte{inactivePubKey[:], activePubKey[:]},
			},
		).Return(activeClientStream, nil)
		activeClientStream.EXPECT().Recv().Return(
			activeResp,
			nil,
		)

		channel := make(chan [][fieldparams.BLSPubkeyLength]byte)
		go func() {
			// We add the active key into the keymanager and simulate a key refresh.
			time.Sleep(time.Second * 1)
			err = km.RecoverAccountsFromMnemonic(ctx, constant.TestMnemonic, derived.DefaultMnemonicLanguage, "", 2)
			require.NoError(t, err)
			channel <- [][fieldparams.BLSPubkeyLength]byte{}
		}()

		assert.NoError(t, v.internalWaitForActivation(context.Background(), channel))
		assert.LogsContain(t, hook, "Waiting for deposit to be observed by beacon node")
		assert.LogsContain(t, hook, "Validator activated")
	})
}

func TestWaitActivation_NotAllValidatorsActivatedOK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	validatorClient := validatormock.NewMockValidatorClient(ctrl)
	chainClient := validatormock.NewMockChainClient(ctrl)
	prysmChainClient := validatormock.NewMockPrysmChainClient(ctrl)

	kp := randKeypair(t)
	v := validator{
		validatorClient:  validatorClient,
		km:               newMockKeymanager(t, kp),
		chainClient:      chainClient,
		prysmChainClient: prysmChainClient,
		pubkeyToStatus:   make(map[[48]byte]*validatorStatus),
	}
	resp := generateMockStatusResponse([][]byte{kp.pub[:]})
	resp.Statuses[0].Status.Status = ethpb.ValidatorStatus_ACTIVE
	clientStream := mock.NewMockBeaconNodeValidator_WaitForActivationClient(ctrl)
	validatorClient.EXPECT().WaitForActivation(
		gomock.Any(),
		gomock.Any(),
	).Return(clientStream, nil)
	prysmChainClient.EXPECT().ValidatorCount(
		gomock.Any(),
		"head",
		[]validatorType.Status{validatorType.Active},
	).Return([]iface.ValidatorCount{}, nil).Times(2)
	clientStream.EXPECT().Recv().Return(
		&ethpb.ValidatorActivationResponse{},
		nil,
	)
	clientStream.EXPECT().Recv().Return(
		resp,
		nil,
	)
	assert.NoError(t, v.WaitForActivation(context.Background(), nil), "Could not wait for activation")
}
