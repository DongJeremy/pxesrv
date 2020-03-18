%global provider        github
%global provider_tld    com
%global project         DongJeremy
%global repo            pxesrv
# https://github.com/DongJeremy/pxesrv
%global provider_prefix %{provider}.%{provider_tld}/%{project}/%{repo}
%global import_path     %{provider_prefix}

Name:           pxesrv
Version:        0.0.1
Release:        1%{?dist}
Summary:        gonews Daily News Retrieval Platform

Group:          Development/Tools
License:        MIT
URL:            https://%{provider_prefix}
Source0:        https://%{provider_prefix}/archive/v%{version}.tar.gz

BuildRequires:  golang, upx
Requires:       redis, git

%description
Gonews for Daily News Retrieval Platform

%prep
%setup -q


%build
go build -ldflags "-s -w" -o pxesrv main.go
upx --brute pxesrv


%install
install -d -p %{buildroot}/usr/local/%{name}/
install -p -m 0755 pxesrv %{buildroot}/usr/local/%{name}/%{name}
install -p -m 0644 pxe.yml %{buildroot}/usr/local/%{name}/
cp -a netboot/ %{buildroot}/usr/local/%{name}/

mkdir -p $RPM_BUILD_ROOT%{_unitdir}
install -m 0644 %{name}.service $RPM_BUILD_ROOT%{_unitdir}


%files
%defattr(-,root,root)
/usr/local/%{name}
%{_unitdir}/%{name}.service


%changelog
