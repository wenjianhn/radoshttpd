%global debug_package %{nil}
%global __strip /bin/true

Name:	    wuzei
Version:	%{ver}
Release:	%{rel}%{?dist}
Summary:	HTTP server for ceph

Group:		System Environment/Base
License:	GPL
URL:		http://10.150.130.22:22222/ceph/radoshttpd
Source0:	 %{name}-%{version}-%{rel}.tar.gz
BuildRoot:	%(mktemp -ud %{_tmppath}/%{name}-%{version}-%{release}-XXXXXX)

BuildRequires:	ceph-devel
Requires:	libradosstriper1

%description
A lightweight HTTP server to obtain ceph's striped object. Only
support download. 


%prep
%setup -q -n %{name}-%{version}-%{rel}


%build
make %{?_smp_mflags}



%install
rm -rf %{buildroot}
make install DESTDIR=%{buildroot}


%clean
rm -rf %{buildroot}


%files
%defattr(-,root,root,-)
%{_bindir}/wuzei
/etc/init.d/wuzei
/etc/logrotate.d/wuzei
%doc



%changelog

