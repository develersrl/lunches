package tinabot

import (
	"testing"

	"github.com/develersrl/lunches/actions/brain"
	"github.com/develersrl/lunches/pkg/tuttobene"
)

func TestOrder(t *testing.T) {

	order := NewOrder()

	p := tuttobene.MenuRow{
		Content: "primo",
		Type:    tuttobene.Primo,
	}
	s := tuttobene.MenuRow{
		Content: "secondo",
		Type:    tuttobene.Secondo,
	}
	s2 := tuttobene.MenuRow{
		Content: "secondo2",
		Type:    tuttobene.Secondo,
	}

	var uc1, uc2, uc3 UserChoice
	uc1.Add(p)
	uc2.Add(s)
	uc3.Add(s2)
	uclist := []UserChoice{uc1, uc2}
	uclist2 := []UserChoice{uc3}
	order.Set("test", uclist)
	assertEqual(t, order.String(), "1 primo [test]\n1 secondo [test]", "")
	assertEqual(t, order.Format(false), "1 primo\n1 secondo", "")
	order.Set("test2", uclist)
	assertEqual(t, order.String(), "2 primo [test, test2]\n2 secondo [test, test2]", "")
	order.Set("test3", uclist2)
	assertEqual(t, order.String(), "2 primo [test, test2]\n2 secondo [test, test2]\n1 secondo2 [test3]", "")

	o := order.ClearUser("test")
	assertEqual(t, o, "primo\nsecondo", "")
	assertEqual(t, order.String(), "1 primo [test2]\n1 secondo [test2]\n1 secondo2 [test3]", "")
	o = order.ClearUser("test3")
	assertEqual(t, o, "secondo2", "")
	assertEqual(t, order.String(), "1 primo [test2]\n1 secondo [test2]", "")
	b := brain.NewBrainMock()
	e := order.Save(b)
	assertEqual(t, e, nil, "")
	assertEqual(t, len(b), 1, "")
	neworder := NewOrder()
	e = neworder.Load(b)
	assertEqual(t, e, nil, "")
	assertEqual(t, order.String(), neworder.String(), "")
	assertEqual(t, order.Timestamp.Format("2006-01-02T15:04:05.999999-07:00"), neworder.Timestamp.Format("2006-01-02T15:04:05.999999-07:00"), "")
}
