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

	r, err := Compile(MAINTAINER, "ports@freeb", false)
	if err != nil {
		t.Fatal(err)
	}

	for i, x := range matches {
		res := r.re.FindStringSubmatch(x)
		if res == nil {
			t.Errorf("[matches #%d] expected to match %q", i, x)
		}
	}

	for i, x := range nomatches {
		res := r.re.FindStringSubmatch(x)
		if res != nil {
			t.Errorf("[nomatches #%d] expected not to match %q, got %#v", i, x, res)
		}
	}
}

func TestMaintainerRe(t *testing.T) {
	matches := []string{
		"MAINTAINER=	ports@freebsd.org",
		"MAINTAINER=	ports@FreeBSD.org",
	}

	nomatches := []string{
		"MAINTAINER=	xports@freebsd.org",
		"MAINTAINER=	dmgk@freebsd.org",
	}

	r, err := Compile(MAINTAINER, "p.*s@", true)
	if err != nil {
		t.Fatal(err)
	}

	for i, x := range matches {
		res := r.re.FindStringSubmatch(x)
		if res == nil {
			t.Errorf("[matches #%d] expected to match %q", i, x)
		}
	}

	for i, x := range nomatches {
		res := r.re.FindStringSubmatch(x)
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
		"OPT_USES=	go",
	}

	nomatches := []string{
		"USES=	cargo",
		"USES=	gmake golang",
		"USES=	gmake-go xorg",
		"USES=	go-test",
		"USES=	go.test",
		"XUSES=	go",
	}

	r, err := Compile(USES, "go", false)
	if err != nil {
		t.Fatal(err)
	}

	for i, x := range matches {
		res := r.re.FindStringSubmatch(x)
		if res == nil {
			t.Errorf("[matches #%d] expected to match %q", i, x)
		}
	}

	for i, x := range nomatches {
		res := r.re.FindStringSubmatch(x)
		if res != nil {
			t.Errorf("[nomatches #%d] expected not to match %q, got %#v", i, x, res)
		}
	}
}

func TestDepends(t *testing.T) {
	matches := []string{
		"BUILD_DEPENDS=	dash:shells/dash bash:shells/bash",
		"RUN_DEPENDS=	bash>0:shells/bash dash:shells/dash",
		"OPT_DEPENDS=	/usr/local/bin/bash:shells/bash",
	}

	nomatches := []string{
		"BUILD_DEPENDS=	bash-devel:/shells/bash-devel",
		"BUILD_DEPENDS=	other-bash:/shells/other-bash",
		"_DEPENDS=		bash:/shells/bash",
	}

	r, err := Compile(DEPENDS, "bash", false)
	if err != nil {
		t.Fatal(err)
	}

	for i, x := range matches {
		res := r.re.FindStringSubmatch(x)
		if res == nil {
			t.Errorf("[matches #%d] expected to match %q", i, x)
		}
	}

	for i, x := range nomatches {
		res := r.re.FindStringSubmatch(x)
		if res != nil {
			t.Errorf("[nomatches #%d] expected not to match %q, got %#v", i, x, res)
		}
	}
}
