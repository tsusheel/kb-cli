Name:           kb
Version:        0.1.0
Release:        1%{?dist}
Summary:        Knowledge Base CLI

License:        MIT
URL:            https://github.com/tsusheel/kb-cli
Source0:        https://github.com/tsusheel/kb-cli/archive/v%{version}.tar.gz

BuildRequires:  golang

%description
Knowledge Base CLI tool for managing notes and tasks.

%prep
%autosetup -n kb-cli-%{version}

%build
go build -ldflags="-s -w" -o bin/kb main.go

%install
install -D -p -m 0755 bin/kb %{buildroot}%{_bindir}/kb

%files
%license LICENSE
%doc README.md
%{_bindir}/kb
