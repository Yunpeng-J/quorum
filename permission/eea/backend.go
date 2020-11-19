package eea

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	eb "github.com/ethereum/go-ethereum/permission/eea/bind"
	ptype "github.com/ethereum/go-ethereum/permission/types"
)

type Backend struct {
	Ib    ptype.InterfaceBackend
	Contr *Init
}

func (b *Backend) ManageAccountPermissions() error {
	chAccessModified := make(chan *eb.AcctManagerAccountAccessModified)
	chAccessRevoked := make(chan *eb.AcctManagerAccountAccessRevoked)
	chStatusChanged := make(chan *eb.AcctManagerAccountStatusChanged)

	opts := &bind.WatchOpts{}
	var blockNumber uint64 = 1
	opts.Start = &blockNumber

	if _, err := b.Contr.PermAcct.AcctManagerFilterer.WatchAccountAccessModified(opts, chAccessModified); err != nil {
		return fmt.Errorf("failed AccountAccessModified: %v", err)
	}

	if _, err := b.Contr.PermAcct.AcctManagerFilterer.WatchAccountAccessRevoked(opts, chAccessRevoked); err != nil {
		return fmt.Errorf("failed AccountAccessRevoked: %v", err)
	}

	if _, err := b.Contr.PermAcct.AcctManagerFilterer.WatchAccountStatusChanged(opts, chStatusChanged); err != nil {
		return fmt.Errorf("failed AccountStatusChanged: %v", err)
	}

	go func() {
		stopChan, stopSubscription := ptype.SubscribeStopEvent()
		defer stopSubscription.Unsubscribe()
		for {
			select {
			case evtAccessModified := <-chAccessModified:
				types.AcctInfoMap.UpsertAccount(evtAccessModified.OrgId, evtAccessModified.RoleId, evtAccessModified.Account, evtAccessModified.OrgAdmin, types.AcctStatus(int(evtAccessModified.Status.Uint64())))

			case evtAccessRevoked := <-chAccessRevoked:
				types.AcctInfoMap.UpsertAccount(evtAccessRevoked.OrgId, evtAccessRevoked.RoleId, evtAccessRevoked.Account, evtAccessRevoked.OrgAdmin, types.AcctActive)

			case evtStatusChanged := <-chStatusChanged:
				if ac, err := types.AcctInfoMap.GetAccount(evtStatusChanged.Account); ac != nil {
					types.AcctInfoMap.UpsertAccount(evtStatusChanged.OrgId, ac.RoleId, evtStatusChanged.Account, ac.IsOrgAdmin, types.AcctStatus(int(evtStatusChanged.Status.Uint64())))
				} else {
					log.Info("error fetching account information", "err", err)
				}
			case <-stopChan:
				log.Info("quit account contract watch")
				return
			}
		}
	}()
	return nil
}

func (b *Backend) ManageRolePermissions() error {
	chRoleCreated := make(chan *eb.RoleManagerRoleCreated, 1)
	chRoleRevoked := make(chan *eb.RoleManagerRoleRevoked, 1)

	opts := &bind.WatchOpts{}
	var blockNumber uint64 = 1
	opts.Start = &blockNumber

	if _, err := b.Contr.PermRole.RoleManagerFilterer.WatchRoleCreated(opts, chRoleCreated); err != nil {
		return fmt.Errorf("failed WatchRoleCreated: %v", err)
	}

	if _, err := b.Contr.PermRole.RoleManagerFilterer.WatchRoleRevoked(opts, chRoleRevoked); err != nil {
		return fmt.Errorf("failed WatchRoleRevoked: %v", err)
	}

	go func() {
		stopChan, stopSubscription := ptype.SubscribeStopEvent()
		defer stopSubscription.Unsubscribe()
		for {
			select {
			case evtRoleCreated := <-chRoleCreated:
				types.RoleInfoMap.UpsertRole(evtRoleCreated.OrgId, evtRoleCreated.RoleId, evtRoleCreated.IsVoter, evtRoleCreated.IsAdmin, types.AccessType(int(evtRoleCreated.BaseAccess.Uint64())), true)

			case evtRoleRevoked := <-chRoleRevoked:
				if r, _ := types.RoleInfoMap.GetRole(evtRoleRevoked.OrgId, evtRoleRevoked.RoleId); r != nil {
					types.RoleInfoMap.UpsertRole(evtRoleRevoked.OrgId, evtRoleRevoked.RoleId, r.IsVoter, r.IsAdmin, r.Access, false)
				} else {
					log.Error("Revoke role - cache is missing role", "org", evtRoleRevoked.OrgId, "role", evtRoleRevoked.RoleId)
				}
			case <-stopChan:
				log.Info("quit role contract watch")
				return
			}
		}
	}()
	return nil
}

func (b *Backend) ManageOrgPermissions() error {
	chPendingApproval := make(chan *eb.OrgManagerOrgPendingApproval, 1)
	chOrgApproved := make(chan *eb.OrgManagerOrgApproved, 1)
	chOrgSuspended := make(chan *eb.OrgManagerOrgSuspended, 1)
	chOrgReactivated := make(chan *eb.OrgManagerOrgSuspensionRevoked, 1)

	opts := &bind.WatchOpts{}
	var blockNumber uint64 = 1
	opts.Start = &blockNumber

	if _, err := b.Contr.PermOrg.OrgManagerFilterer.WatchOrgPendingApproval(opts, chPendingApproval); err != nil {
		return fmt.Errorf("failed WatchOrgPendingApproval: %v", err)
	}

	if _, err := b.Contr.PermOrg.OrgManagerFilterer.WatchOrgApproved(opts, chOrgApproved); err != nil {
		return fmt.Errorf("failed WatchOrgApproved: %v", err)
	}

	if _, err := b.Contr.PermOrg.OrgManagerFilterer.WatchOrgSuspended(opts, chOrgSuspended); err != nil {
		return fmt.Errorf("failed WatchOrgSuspended: %v", err)
	}

	if _, err := b.Contr.PermOrg.OrgManagerFilterer.WatchOrgSuspensionRevoked(opts, chOrgReactivated); err != nil {
		return fmt.Errorf("failed WatchOrgSuspensionRevoked: %v", err)
	}

	go func() {
		stopChan, stopSubscription := ptype.SubscribeStopEvent()
		defer stopSubscription.Unsubscribe()
		for {
			select {
			case evtPendingApproval := <-chPendingApproval:
				types.OrgInfoMap.UpsertOrg(evtPendingApproval.OrgId, evtPendingApproval.PorgId, evtPendingApproval.UltParent, evtPendingApproval.Level, types.OrgStatus(evtPendingApproval.Status.Uint64()))

			case evtOrgApproved := <-chOrgApproved:
				types.OrgInfoMap.UpsertOrg(evtOrgApproved.OrgId, evtOrgApproved.PorgId, evtOrgApproved.UltParent, evtOrgApproved.Level, types.OrgApproved)

			case evtOrgSuspended := <-chOrgSuspended:
				types.OrgInfoMap.UpsertOrg(evtOrgSuspended.OrgId, evtOrgSuspended.PorgId, evtOrgSuspended.UltParent, evtOrgSuspended.Level, types.OrgSuspended)

			case evtOrgReactivated := <-chOrgReactivated:
				types.OrgInfoMap.UpsertOrg(evtOrgReactivated.OrgId, evtOrgReactivated.PorgId, evtOrgReactivated.UltParent, evtOrgReactivated.Level, types.OrgApproved)
			case <-stopChan:
				log.Info("quit org contract watch")
				return
			}
		}
	}()
	return nil
}

func (b *Backend) ManageNodePermissions() error {
	chNodeApproved := make(chan *eb.NodeManagerNodeApproved, 1)
	chNodeProposed := make(chan *eb.NodeManagerNodeProposed, 1)
	chNodeDeactivated := make(chan *eb.NodeManagerNodeDeactivated, 1)
	chNodeActivated := make(chan *eb.NodeManagerNodeActivated, 1)
	chNodeBlacklisted := make(chan *eb.NodeManagerNodeBlacklisted)
	chNodeRecoveryInit := make(chan *eb.NodeManagerNodeRecoveryInitiated, 1)
	chNodeRecoveryDone := make(chan *eb.NodeManagerNodeRecoveryCompleted, 1)

	opts := &bind.WatchOpts{}
	var blockNumber uint64 = 1
	opts.Start = &blockNumber

	if _, err := b.Contr.PermNode.NodeManagerFilterer.WatchNodeApproved(opts, chNodeApproved); err != nil {
		return fmt.Errorf("failed WatchNodeApproved: %v", err)
	}

	if _, err := b.Contr.PermNode.NodeManagerFilterer.WatchNodeProposed(opts, chNodeProposed); err != nil {
		return fmt.Errorf("failed WatchNodeProposed: %v", err)
	}

	if _, err := b.Contr.PermNode.NodeManagerFilterer.WatchNodeDeactivated(opts, chNodeDeactivated); err != nil {
		return fmt.Errorf("failed NodeDeactivated: %v", err)
	}
	if _, err := b.Contr.PermNode.NodeManagerFilterer.WatchNodeActivated(opts, chNodeActivated); err != nil {
		return fmt.Errorf("failed WatchNodeActivated: %v", err)
	}

	if _, err := b.Contr.PermNode.NodeManagerFilterer.WatchNodeBlacklisted(opts, chNodeBlacklisted); err != nil {
		return fmt.Errorf("failed NodeBlacklisting: %v", err)
	}

	if _, err := b.Contr.PermNode.NodeManagerFilterer.WatchNodeRecoveryInitiated(opts, chNodeRecoveryInit); err != nil {
		return fmt.Errorf("failed NodeRecoveryInitiated: %v", err)
	}

	if _, err := b.Contr.PermNode.NodeManagerFilterer.WatchNodeRecoveryCompleted(opts, chNodeRecoveryDone); err != nil {
		return fmt.Errorf("failed NodeRecoveryCompleted: %v", err)
	}

	go func() {
		stopChan, stopSubscription := ptype.SubscribeStopEvent()
		defer stopSubscription.Unsubscribe()
		for {
			select {
			case evtNodeApproved := <-chNodeApproved:
				enodeId := types.GetNodeUrl(evtNodeApproved.EnodeId, evtNodeApproved.Ip[:], evtNodeApproved.Port, evtNodeApproved.Raftport, b.Ib.IsRaft())
				err := ptype.UpdatePermissionedNodes(b.Ib.Node(), b.Ib.DataDir(), enodeId, ptype.NodeAdd, b.Ib.IsRaft())
				if err != nil {
					log.Error("error updating permissioned-nodes.json", "err", err)
				}
				types.NodeInfoMap.UpsertNode(evtNodeApproved.OrgId, enodeId, types.NodeApproved)

			case evtNodeProposed := <-chNodeProposed:
				enodeId := types.GetNodeUrl(evtNodeProposed.EnodeId, evtNodeProposed.Ip[:], evtNodeProposed.Port, evtNodeProposed.Raftport, b.Ib.IsRaft())
				types.NodeInfoMap.UpsertNode(evtNodeProposed.OrgId, enodeId, types.NodePendingApproval)

			case evtNodeDeactivated := <-chNodeDeactivated:
				enodeId := types.GetNodeUrl(evtNodeDeactivated.EnodeId, evtNodeDeactivated.Ip[:], evtNodeDeactivated.Port, evtNodeDeactivated.Raftport, b.Ib.IsRaft())
				err := ptype.UpdatePermissionedNodes(b.Ib.Node(), b.Ib.DataDir(), enodeId, ptype.NodeDelete, b.Ib.IsRaft())
				if err != nil {
					log.Error("error updating permissioned-nodes.json", "err", err)
				}
				types.NodeInfoMap.UpsertNode(evtNodeDeactivated.OrgId, enodeId, types.NodeDeactivated)

			case evtNodeActivated := <-chNodeActivated:
				enodeId := types.GetNodeUrl(evtNodeActivated.EnodeId, evtNodeActivated.Ip[:], evtNodeActivated.Port, evtNodeActivated.Raftport, b.Ib.IsRaft())
				err := ptype.UpdatePermissionedNodes(b.Ib.Node(), b.Ib.DataDir(), enodeId, ptype.NodeAdd, b.Ib.IsRaft())
				if err != nil {
					log.Error("error updating permissioned-nodes.json", "err", err)
				}
				types.NodeInfoMap.UpsertNode(evtNodeActivated.OrgId, enodeId, types.NodeApproved)

			case evtNodeBlacklisted := <-chNodeBlacklisted:
				enodeId := types.GetNodeUrl(evtNodeBlacklisted.EnodeId, evtNodeBlacklisted.Ip[:], evtNodeBlacklisted.Port, evtNodeBlacklisted.Raftport, b.Ib.IsRaft())
				types.NodeInfoMap.UpsertNode(evtNodeBlacklisted.OrgId, enodeId, types.NodeBlackListed)
				err := ptype.UpdateDisallowedNodes(b.Ib.DataDir(), enodeId, ptype.NodeAdd)
				log.Error("error updating disallowed-nodes.json", "err", err)
				err = ptype.UpdatePermissionedNodes(b.Ib.Node(), b.Ib.DataDir(), enodeId, ptype.NodeDelete, b.Ib.IsRaft())
				if err != nil {
					log.Error("error updating permissioned-nodes.json", "err", err)
				}

			case evtNodeRecoveryInit := <-chNodeRecoveryInit:
				enodeId := types.GetNodeUrl(evtNodeRecoveryInit.EnodeId, evtNodeRecoveryInit.Ip[:], evtNodeRecoveryInit.Port, evtNodeRecoveryInit.Raftport, b.Ib.IsRaft())
				types.NodeInfoMap.UpsertNode(evtNodeRecoveryInit.OrgId, enodeId, types.NodeRecoveryInitiated)

			case evtNodeRecoveryDone := <-chNodeRecoveryDone:
				enodeId := types.GetNodeUrl(evtNodeRecoveryDone.EnodeId, evtNodeRecoveryDone.Ip[:], evtNodeRecoveryDone.Port, evtNodeRecoveryDone.Raftport, b.Ib.IsRaft())
				types.NodeInfoMap.UpsertNode(evtNodeRecoveryDone.OrgId, enodeId, types.NodeApproved)
				err := ptype.UpdateDisallowedNodes(b.Ib.DataDir(), enodeId, ptype.NodeDelete)
				log.Error("error updating disallowed-nodes.json", "err", err)
				err = ptype.UpdatePermissionedNodes(b.Ib.Node(), b.Ib.DataDir(), enodeId, ptype.NodeAdd, b.Ib.IsRaft())
				if err != nil {
					log.Error("error updating permissioned-nodes.json", "err", err)
				}

			case <-stopChan:
				log.Info("quit Node contract watch")
				return
			}
		}
	}()
	return nil
}
func (b *Backend) MonitorNetworkBootUp() error {
	return nil
}

func getBackend(contractBackend ptype.ContractBackend) (*Eea, error) {
	eeaBackend := Eea{ContractBackend: contractBackend}
	ps, err := getInterfaceContractSession(eeaBackend.PermInterf, contractBackend.PermConfig.InterfAddress, contractBackend.EthClnt)
	if err != nil {
		return nil, err
	}
	eeaBackend.PermInterfSession = ps
	return &eeaBackend, nil
}

func getBackendWithTransactOpts(contractBackend ptype.ContractBackend, transactOpts *bind.TransactOpts) (*Eea, error) {
	eeaBackend, err := getBackend(contractBackend)
	if err != nil {
		return nil, err
	}
	eeaBackend.PermInterfSession.TransactOpts = *transactOpts
	return eeaBackend, nil
}

func getInterfaceContractSession(permInterfaceInstance *eb.PermInterface, contractAddress common.Address, backend bind.ContractBackend) (*eb.PermInterfaceSession, error) {
	if err := ptype.BindContract(&permInterfaceInstance, func() (interface{}, error) { return eb.NewPermInterface(contractAddress, backend) }); err != nil {
		return nil, err
	}
	ps := &eb.PermInterfaceSession{
		Contract: permInterfaceInstance,
		CallOpts: bind.CallOpts{
			Pending: true,
		},
	}
	return ps, nil
}

func (b *Backend) GetRoleService(transactOpts *bind.TransactOpts, roleBackend ptype.ContractBackend) (ptype.RoleService, error) {

	backEnd, err := getBackendWithTransactOpts(roleBackend, transactOpts)
	if err != nil {
		return nil, err
	}
	return &Role{Backend: backEnd}, nil

}

func (b *Backend) GetOrgService(transactOpts *bind.TransactOpts, orgBackend ptype.ContractBackend) (ptype.OrgService, error) {
	backEnd, err := getBackendWithTransactOpts(orgBackend, transactOpts)
	if err != nil {
		return nil, err
	}
	return &Org{Backend: backEnd}, nil
}

func (b *Backend) GetNodeService(transactOpts *bind.TransactOpts, nodeBackend ptype.ContractBackend) (ptype.NodeService, error) {
	backEnd, err := getBackendWithTransactOpts(nodeBackend, transactOpts)
	if err != nil {
		return nil, err
	}
	return &Node{Backend: backEnd}, nil
}

func (b *Backend) GetAccountService(transactOpts *bind.TransactOpts, accountBackend ptype.ContractBackend) (ptype.AccountService, error) {
	backEnd, err := getBackendWithTransactOpts(accountBackend, transactOpts)
	if err != nil {
		return nil, err
	}
	return &Account{Backend: backEnd}, nil

}

func (b *Backend) GetAuditService(auditBackend ptype.ContractBackend) (ptype.AuditService, error) {
	backEnd, err := getBackend(auditBackend)
	if err != nil {
		return nil, err
	}
	return &Audit{Backend: backEnd}, nil

}

func (b *Backend) GetControlService(controlBackend ptype.ContractBackend) (ptype.ControlService, error) {
	backEnd, err := getBackend(controlBackend)
	if err != nil {
		return nil, err
	}
	return &Control{Backend: backEnd}, nil

}
