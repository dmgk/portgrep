package grep

import (
	"testing"
)

func TestMaintainer(t *testing.T) {
	matches := []string{
		"MAINTAINER=ports@freebsd.org",
		"MAINTAINER=	ports@FreeBSD.org",
		"MAINTAINER=	ports@freebsd",
	}

	nomatches := []string{
		"MAINTAINER=xports@freebsd.org",
		"MAINTAINER=	xports@freebsd.org",
		"XMAINTAINER=	ports",
		"_MAINTAINER=	ports",
	}

	re, err := Compile(MAINTAINER, "ports@freeb")
	if err != nil {
		t.Fatal(err)
	}

	for i, x := range matches {
		res := re.FindStringSubmatch(x)
		if res == nil {
			t.Errorf("[matches #%d] expected to match %q", i, x)
		}
	}

	for i, x := range nomatches {
		res := re.FindStringSubmatch(x)
		if res != nil {
			t.Errorf("[nomatches #%d] expected not to match %q, got %#v", i, x, res)
		}
	}
}

func TestUses(t *testing.T) {
	matches := []string{
		"USES=go",
		"USES=	go",
		"USES=	go xorg",
		"USES=	gmake go xorg",
		"USES=	gmake go",
		"_USES=	go",
	}

	nomatches := []string{
		"USES=	cargo",
		"USES=	gmake golang",
		"USES=	gmake-go xorg",
		"USES=	go-test",
		"USES=	go.test",
		"XUSES=	go",
	}

	re, err := Compile(USES, "go")
	if err != nil {
		t.Fatal(err)
	}

	for i, x := range matches {
		res := re.FindStringSubmatch(x)
		if res == nil {
			t.Errorf("[matches #%d] expected to match %q", i, x)
		}
	}

	for i, x := range nomatches {
		res := re.FindStringSubmatch(x)
		if res != nil {
			t.Errorf("[nomatches #%d] expected not to match %q, got %#v", i, x, res)
		}
	}
}
