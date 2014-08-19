Name:   msgclient
Summary:        A decentralized and distributed message synchronization system
Version:        1.0.1
Release:        el5

Group:          Applications/Server
License:        GPLv2
URL:            https://github.com/fangli/gomsgfiber
Source0:        https://github.com/fangli/gomsgfiber/releases/download/1.0.1/msgclient.tar.gz
BuildRoot:      %(mktemp -ud %{_tmppath}/%{name}-%{version}-%{release}-XXXXXX)

%define debug_package %{nil}

%description
A decentralized and distributed message synchronization system

%prep
%setup -c

%pre

%install
install -d %{buildroot}%{_bindir}
mv msgclient %{buildroot}%{_bindir}/
mv msgclient-manager %{buildroot}%{_bindir}/
install -d %{buildroot}%{_initrddir}
mv etc/init.d/msgclient %{buildroot}%{_initrddir}/
install -d %{buildroot}%{_sysconfdir}
mv etc/msgclient %{buildroot}%{_sysconfdir}/


%clean
rm -rf %{buildroot}

%files
%defattr(-,root,root,-)
%{_bindir}/*
%{_initrddir}/*
%config %{_sysconfdir}/*

%post
/sbin/chkconfig --add msgclient

%preun
  if [ $1 = 0 ]; then
      # package is being erased, not upgraded
      /sbin/chkconfig --del msgclient
  fi

%postun
  if [ $1 = 0 ]; then
      echo "Uninstalling msgclient..."
      # package is being erased
      # Any needed actions here on uninstalls
  else
      # Upgrade
      echo "Msgclient stopped, do not forget to restart it again"
  fi


%changelog
* Mon Aug 18 2014  Fang.li <fang.li@funplus.com>
  - fixed CentOS 5 compatiable bugs
  - Initial release
