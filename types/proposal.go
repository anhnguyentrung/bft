package types

type View struct {
	Round 		uint64
	Height 		uint64
}

func (v View) Next() View {
	return View{
		Round: v.Round + 1,
		Height: v.Height + 1,
	}
}

func (v View) Compare(target View) int {
	if v.Height < target.Height {
		return -1
	}
	if v.Height > target.Height {
		return 1
	}
	if v.Round < target.Round {
		return -1
	}
	if v.Round > target.Round {
		return 1
	}
	return 0
}

type Proposal struct {
	View 	View
	Sender  Validator
	Block 	Block
}

func (p *Proposal) BlockId() Hash {
	return p.Block.Header().Id()
}

func (p *Proposal) BlockHeightId() BlockHeightId {
	return p.Block.Header().HeightId
}

func (p *Proposal) Proposer() Validator {
	return p.Block.Header().Proposer
}