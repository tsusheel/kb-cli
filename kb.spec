%global debug_package %{nil}

Name:           kb
Version:        0.1.3
Release:        1%{?dist}
Summary:        Knowledge Base CLI

License:        MIT
URL:            https://github.com/tsusheel/kb-cli
Source0:        https://github.com/tsusheel/kb-cli/archive/v%{version}.tar.gz

BuildRequires:  golang
BuildRequires:  git

%description
Knowledge Base CLI tool for managing notes and tasks.

%prep
%autosetup -n kb-cli-%{version}

%build
go build -ldflags="-s -w" -o bin/kb main.go

%install
install -D -p -m 0755 bin/kb %{buildroot}%{_bindir}/kb

%post
if [ $1 -ge 2 ]; then
    if command -v systemctl >/dev/null 2>&1; then
        systemctl try-restart kb.service >/dev/null 2>&1 || true
    fi
fi

%files
%license LICENSE
%doc README.md
%{_bindir}/kb

%changelog
* Sun Oct 04 2026 Sushil Thakur <tsusheel.135@gmail.com> - 0.1.3-1
- Implement incremental delta sync and batch entity synchronization
- Add parallel entity fetch queries and atomic transaction processing
- Add comprehensive unit tests for sync optimizations

* Sun Oct 04 2026 Sushil Thakur <tsusheel.135@gmail.com> - 0.1.2-1
- Add Cache-Control and CDN anti-stale headers for static assets
- Add post-upgrade service restart trigger

* Sun Oct 04 2026 Sushil Thakur <tsusheel.135@gmail.com> - 0.1.1-1
- Add web UI authentication, session persistence, and logout support

* Sun Oct 04 2026 Sushil Thakur <tsusheel.135@gmail.com> - 0.1.0-1
- Initial package release

