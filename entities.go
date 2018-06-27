package demoinfocs

import (
	"bytes"

	bit "github.com/markus-wa/demoinfocs-golang/bitread"
	msg "github.com/markus-wa/demoinfocs-golang/msg"
	st "github.com/markus-wa/demoinfocs-golang/sendtables"
)

const entitySentinel = 9999

func (p *Parser) handlePacketEntities(pe *msg.CSVCMsg_PacketEntities) {
	defer func() {
		p.setError(recoverFromUnexpectedEOF(recover()))
	}()

	r := bit.NewSmallBitReader(bytes.NewReader(pe.EntityData))

	currentEntity := -1
	for i := 0; i < int(pe.UpdatedEntries); i++ {
		currentEntity += 1 + int(r.ReadUBitInt())

		if currentEntity > entitySentinel {
			break
		}

		if r.ReadBit() {
			// Leave PVS

			// FIXME: Might have to destroy the entities contents first, not sure yet
			// Could do weird stuff with event handlers otherwise
			p.entities[currentEntity] = nil

			// TODO: Force Delete if this is true??
			r.ReadBit()
		} else {
			if r.ReadBit() {
				// Enter PVS
				e := p.readEnterPVS(r, currentEntity)
				p.entities[currentEntity] = e
				e.ApplyUpdate(r)
				e.ServerClass.FireEntityCreatedEvent(e)
			} else {
				// Delta Update
				p.entities[currentEntity].ApplyUpdate(r)
			}
		}
	}
	r.Pool()
}

func (p *Parser) readEnterPVS(reader *bit.BitReader, entityID int) *st.Entity {
	scID := int(reader.ReadInt(p.stParser.ClassBits()))
	reader.Skip(10) // Serial Number

	newEntity := st.NewEntity(entityID, p.stParser.ServerClasses()[scID])

	if p.preprocessedBaselines[scID] != nil {
		for idx, val := range p.preprocessedBaselines[scID] {
			newEntity.Props()[idx].FirePropertyUpdate(val)
		}
	} else {
		if p.instanceBaselines[scID] != nil {
			r := bit.NewSmallBitReader(bytes.NewReader(p.instanceBaselines[scID]))
			p.preprocessedBaselines[scID] = newEntity.InitializeBaseline(r)
			r.Pool()
		} else {
			p.preprocessedBaselines[scID] = make(map[int]st.PropValue)
		}
	}

	return newEntity
}
