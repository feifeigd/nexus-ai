package logic

import (
	"nexusai/services/api/internal/data"
	"nexusai/services/api/internal/types"
)

func machineResponse(m data.MachineRecord) types.Machine {
	return types.Machine{
		ID:             m.ID,
		Name:           m.Name,
		PrivateAddress: m.PrivateAddress,
		PublicAddress:  m.PublicAddress,
		Region:         m.Region,
		RootPublicKey:  m.RootPublicKey,
		JoinedK8s:      m.JoinedK8s,
		CreatedAt:      formatTime(m.CreatedAt),
		UpdatedAt:      formatTime(m.UpdatedAt),
	}
}
