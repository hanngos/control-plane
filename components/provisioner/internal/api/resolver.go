package api

import (
	"context"

	"github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"

	"github.com/kyma-project/control-plane/components/provisioner/internal/api/middlewares"

	log "github.com/sirupsen/logrus"

	"github.com/kyma-project/control-plane/components/provisioner/internal/provisioning"
	"github.com/kyma-project/control-plane/components/provisioner/pkg/gqlschema"
)

type Resolver struct {
	provisioning provisioning.Service
	validator    Validator
}

func (r *Resolver) Mutation() gqlschema.MutationResolver {
	return &Resolver{
		provisioning: r.provisioning,
		validator:    r.validator,
	}
}
func (r *Resolver) Query() gqlschema.QueryResolver {
	return &Resolver{
		provisioning: r.provisioning,
		validator:    r.validator,
	}
}

func NewResolver(provisioningService provisioning.Service, validator Validator) *Resolver {
	return &Resolver{
		provisioning: provisioningService,
		validator:    validator,
	}
}

func (r *Resolver) ProvisionRuntime(ctx context.Context, config gqlschema.ProvisionRuntimeInput) (*gqlschema.OperationStatus, error) {
	err := r.validator.ValidateProvisioningInput(config)
	if err != nil {
		log.Errorf("Failed to provision Runtime %s", err)
		return nil, err
	}

	tenant, err := getTenant(ctx)
	if err != nil {
		log.Errorf("Failed to provision Runtime %s: %s", config.RuntimeInput.Name, err)
		return nil, err
	}

	subAccount := getSubAccount(ctx)

	log.Infof("Requested provisioning of Runtime %s.", config.RuntimeInput.Name)

	operationStatus, err := r.provisioning.ProvisionRuntime(config, tenant, subAccount)
	if err != nil {
		log.Errorf("Failed to provision Runtime %s: %s", config.RuntimeInput.Name, err)
		return nil, err
	}
	log.Infof("Provisioning started for Runtime %s. Operation id %s", config.RuntimeInput.Name, *operationStatus.ID)

	return operationStatus, nil
}

func (r *Resolver) DeprovisionRuntime(ctx context.Context, id string) (string, error) {
	log.Infof("Requested deprovisioning of Runtime %s.", id)

	_, err := r.getAndValidateTenant(ctx, id)
	if err != nil {
		log.Errorf("Failed to deprovision Runtime %s: %s", id, err)
		return "", err
	}

	operationID, err := r.provisioning.DeprovisionRuntime(id)
	if err != nil {
		log.Errorf("Failed to deprovision Runtime %s: %s", id, err)
		return "", err
	}
	log.Infof("Deprovisioning started for Runtime %s. Operation id %s", id, operationID)

	return operationID, nil
}

func (r *Resolver) UpgradeRuntime(ctx context.Context, runtimeId string, input gqlschema.UpgradeRuntimeInput) (*gqlschema.OperationStatus, error) {
	log.Infof("Requested upgrade of Runtime %s.", runtimeId)

	if _, err := r.getAndValidateTenant(ctx, runtimeId); err != nil {
		log.Errorf("Failed to upgrade Runtime %s: %s", runtimeId, err)
		return &gqlschema.OperationStatus{}, err
	}

	if err := r.validator.ValidateUpgradeInput(input); err != nil {
		log.Errorf("Failed to upgrade Runtime %s: %s", runtimeId, err)
		return nil, err
	}

	operationStatus, err := r.provisioning.UpgradeRuntime(runtimeId, input)
	if err != nil {
		log.Errorf("Failed to upgrade Runtime %s: %s", runtimeId, err)
		return nil, err
	}

	return operationStatus, nil
}

func (r *Resolver) RollBackUpgradeOperation(ctx context.Context, runtimeID string) (*gqlschema.RuntimeStatus, error) {
	_, err := r.getAndValidateTenant(ctx, runtimeID)
	if err != nil {
		log.Errorf("Failed to roll back last Runtime upgrade: %s, Runtime ID: %s", err, runtimeID)
		return nil, err
	}

	runtimeStatus, err := r.provisioning.RollBackLastUpgrade(runtimeID)
	if err != nil {
		log.Errorf("Failed to roll back last Runtime upgrade: %s, Runtime ID: %s", err, runtimeID)
		return nil, err
	}

	return runtimeStatus, nil
}

func (r *Resolver) ReconnectRuntimeAgent(ctx context.Context, id string) (string, error) {
	return "", nil
}

func (r *Resolver) RuntimeStatus(ctx context.Context, runtimeID string) (*gqlschema.RuntimeStatus, error) {
	log.Infof("Requested to get status for Runtime %s.", runtimeID)

	_, err := r.getAndValidateTenant(ctx, runtimeID)
	if err != nil {
		log.Errorf("Failed to get status for Runtime %s: %s", runtimeID, err)
		return nil, err
	}

	status, err := r.provisioning.RuntimeStatus(runtimeID)
	if err != nil {
		log.Errorf("Failed to get status for Runtime %s: %s", runtimeID, err)
		return nil, err
	}
	log.Infof("Getting status for Runtime %s succeeded.", runtimeID)

	return status, nil
}

func (r *Resolver) RuntimeOperationStatus(ctx context.Context, operationID string) (*gqlschema.OperationStatus, error) {
	log.Infof("Requested to get Runtime operation status for Operation %s.", operationID)

	_, err := r.getAndValidateTenantForOp(ctx, operationID)
	if err != nil {
		log.Errorf("Failed to get Runtime operation status: %s, Operation ID: %s", err, operationID)
		return nil, err
	}

	status, err := r.provisioning.RuntimeOperationStatus(operationID)
	if err != nil {
		log.Errorf("Failed to get Runtime operation status: %s Operation ID: %s", err, operationID)
		return nil, err
	}

	log.Infof("Getting Runtime operation status for Operation %s succeeded.", operationID)

	return status, nil
}

func (r *Resolver) UpgradeShoot(ctx context.Context, runtimeID string, input gqlschema.UpgradeShootInput) (*gqlschema.OperationStatus, error) {
	log.Infof("Requested to upgrade Gardener Shoot cluster specification for Runtime : %s.", runtimeID)

	_, err := r.getAndValidateTenant(ctx, runtimeID)
	if err != nil {
		log.Errorf("Failed to upgrade Gardener Shoot cluster specification for Runtime  %s: %s", runtimeID, err)
		return nil, err
	}

	err = r.validator.ValidateUpgradeShootInput(input)
	if err != nil {
		log.Errorf("Failed to upgrade Gardener Shoot cluster specification for Runtime %s", err)
		return nil, err
	}

	status, err := r.provisioning.UpgradeGardenerShoot(runtimeID, input)
	if err != nil {
		log.Errorf("Failed to upgrade Gardener Shoot cluster specification for Runtime %s: %s", runtimeID, err)
		return nil, err
	}

	log.Infof("Upgrade Gardener Shoot cluster specification for Runtime %s succeeded", runtimeID)

	return status, nil
}

func (r *Resolver) HibernateRuntime(ctx context.Context, runtimeID string) (*gqlschema.OperationStatus, error) {
	log.Infof("Requested to hibernate runtime : %s.", runtimeID)

	_, err := r.getAndValidateTenant(ctx, runtimeID)
	if err != nil {
		log.Errorf("Failed to hibernate Runtime  %s: %s", runtimeID, err)
		return nil, err
	}

	status, err := r.provisioning.HibernateCluster(runtimeID)
	if err != nil {
		log.Errorf("Failed to hibernate Runtime %s: %s", runtimeID, err)
		return nil, err
	}

	return status, nil
}

func (r *Resolver) getAndValidateTenant(ctx context.Context, runtimeID string) (string, error) {
	tenant, err := getTenant(ctx)
	if err != nil {
		return "", err
	}

	err = r.validator.ValidateTenant(runtimeID, tenant)
	if err != nil {
		return "", err
	}

	return tenant, nil
}

func (r *Resolver) getAndValidateTenantForOp(ctx context.Context, operationID string) (string, error) {
	tenant, err := getTenant(ctx)
	if err != nil {
		return "", err
	}

	err = r.validator.ValidateTenantForOperation(operationID, tenant)
	if err != nil {
		return "", err
	}

	return tenant, nil
}

func getTenant(ctx context.Context) (string, apperrors.AppError) {
	tenant, ok := ctx.Value(middlewares.Tenant).(string)
	if !ok || tenant == "" {
		return "", apperrors.BadRequest("tenant header is empty")
	}

	return tenant, nil
}

func getSubAccount(ctx context.Context) string {
	subAccount, ok := ctx.Value(middlewares.SubAccountID).(string)
	if !ok {
		return ""
	}
	return subAccount
}
