package consensus

import (
	"errors"
	"fmt"

	tmcon "github.com/cometbft/cometbft/consensus"
	cstypes "github.com/cometbft/cometbft/consensus/types"
	tmmath "github.com/cometbft/cometbft/libs/math"
	"github.com/cometbft/cometbft/p2p"
	tmcons "github.com/cometbft/cometbft/proto/tendermint/consensus"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cometbft/cometbft/types"
)

func WALToProto(msg tmcon.WALMessage) (*tmcons.WALMessage, error) {
	var pb tmcons.WALMessage

	switch msg := msg.(type) {
	case types.EventDataRoundState:
		pb = tmcons.WALMessage{
			Sum: &tmcons.WALMessage_EventDataRoundState{
				EventDataRoundState: &tmproto.EventDataRoundState{
					Height: msg.Height,
					Round:  msg.Round,
					Step:   msg.Step,
				},
			},
		}
	case msgInfo:
		consMsg, err := tmcon.MsgToProto(msg.Msg)
		if err != nil {
			return nil, err
		}
		pb = tmcons.WALMessage{
			Sum: &tmcons.WALMessage_MsgInfo{
				MsgInfo: &tmcons.MsgInfo{
					Msg:    *consMsg,
					PeerID: string(msg.PeerID),
				},
			},
		}
	case timeoutInfo:
		pb = tmcons.WALMessage{
			Sum: &tmcons.WALMessage_TimeoutInfo{
				TimeoutInfo: &tmcons.TimeoutInfo{
					Duration: msg.Duration,
					Height:   msg.Height,
					Round:    msg.Round,
					Step:     uint32(msg.Step),
				},
			},
		}
	case tmcon.EndHeightMessage:
		pb = tmcons.WALMessage{
			Sum: &tmcons.WALMessage_EndHeight{
				EndHeight: &tmcons.EndHeight{
					Height: msg.Height,
				},
			},
		}
	default:
		return nil, fmt.Errorf("to proto: wal message not recognized: %T", msg)
	}

	return &pb, nil
}

// WALFromProto takes a proto wal message and return a consensus walMessage and error
func WALFromProto(msg *tmcons.WALMessage) (tmcon.WALMessage, error) {
	if msg == nil {
		return nil, errors.New("nil WAL message")
	}
	var pb tmcon.WALMessage

	switch msg := msg.Sum.(type) {
	case *tmcons.WALMessage_EventDataRoundState:
		pb = types.EventDataRoundState{
			Height: msg.EventDataRoundState.Height,
			Round:  msg.EventDataRoundState.Round,
			Step:   msg.EventDataRoundState.Step,
		}
	case *tmcons.WALMessage_MsgInfo:
		walMsg, err := tmcon.MsgFromProto(&msg.MsgInfo.Msg)
		if err != nil {
			return nil, fmt.Errorf("msgInfo from proto error: %w", err)
		}
		pb = msgInfo{
			Msg:    walMsg,
			PeerID: p2p.ID(msg.MsgInfo.PeerID),
		}

	case *tmcons.WALMessage_TimeoutInfo:
		tis, err := tmmath.SafeConvertUint8(int64(msg.TimeoutInfo.Step))
		// deny message based on possible overflow
		if err != nil {
			return nil, fmt.Errorf("denying message due to possible overflow: %w", err)
		}
		pb = timeoutInfo{
			Duration: msg.TimeoutInfo.Duration,
			Height:   msg.TimeoutInfo.Height,
			Round:    msg.TimeoutInfo.Round,
			Step:     cstypes.RoundStepType(tis),
		}
		return pb, nil
	case *tmcons.WALMessage_EndHeight:
		pb := tmcon.EndHeightMessage{
			Height: msg.EndHeight.Height,
		}
		return pb, nil
	default:
		return nil, fmt.Errorf("from proto: wal message not recognized: %T", msg)
	}
	return pb, nil
}
