ARG BASE_IMAGE=ghcr.io/oracle/oraclelinux:8-slim
FROM $BASE_IMAGE as build

# copy in static files
# all scripts are 0755 (rwx r-x r-x)
COPY --chmod=0755 files/usr/local/bin/* /usr/local/bin/

# all configs are 0644 (rw- r-- r--)
COPY --chmod=0644 files/etc/* /etc/
COPY --chmod=0644 files/etc/containers/* /etc/containers/
COPY --chmod=0644 files/etc/crio/crio.conf.d/* /etc/crio/crio.conf.d/
COPY --chmod=0644 files/etc/default/* /etc/default/
COPY --chmod=0644 files/etc/sysctl.d/* /etc/sysctl.d/
COPY --chmod=0644 files/etc/systemd/system/* /etc/systemd/system/
COPY --chmod=0644 files/etc/systemd/system/kubelet.service.d/* /etc/systemd/system/kubelet.service.d/

RUN  microdnf update -y \
     && microdnf install -y  yum  yum-utils \
     && microdnf clean all \
     && rm -rf /var/cache/yum/*


RUN echo "Adding yum repos ..." \
    && yum-config-manager --add-repo https://yum.oracle.com/repo/OracleLinux/OL8/olcne15/x86_64/ \
    && yum-config-manager --add-repo https://yum.oracle.com/repo/OracleLinux/OL8/developer/x86_64/ \
    && yum-config-manager --add-repo https://yum.oracle.com/repo/OracleLinux/OL8/addons/x86_64/ \
    && yum-config-manager --add-repo https://yum.oracle.com/repo/OracleLinux/OL8/UEKR6/x86_64/ \
    && yum-config-manager --add-repo http://yum.oracle.com/repo/OracleLinux/OL8/kvm/appstream/x86_64

RUN echo "Installing Packages ..." \
    && dnf -y install hostname which vi systemd iptables fuse-overlayfs \
    && dnf clean all \
    && rm -rf /var/cache/yum/*


RUN echo "Installing OCNE binaries ..." \
    && dnf install -y  https://yum.oracle.com/repo/OracleLinux/OL8/olcne15/x86_64/getPackage/conmon-2.0.26-2.el8.x86_64.rpm  \
    && dnf install -y  cri-tools cri-o olcne-utils  \
    && dnf install -y kubeadm-1.23.7-1.el8 kubectl-1.23.7-1.el8 kubelet-1.23.7-1.el8 kubernetes-cni-1.0.1-2.el8 kubernetes-cni-plugins-1.0.1-3.el8 \
    && systemctl enable crio.service \
    && dnf clean all \
    && rm -rf /var/cache/yum/*


RUN echo "Prep for systemd ..." \
    && (cd /lib/systemd/system/sysinit.target.wants/; for i in *; do [ $i == \
    systemd-tmpfiles-setup.service ] || rm -f $i; done); \
    rm -f /lib/systemd/system/multi-user.target.wants/*;\
    rm -f /etc/systemd/system/*.wants/*;\
    rm -f /lib/systemd/system/local-fs.target.wants/*; \
    rm -f /lib/systemd/system/sockets.target.wants/*udev*; \
    rm -f /lib/systemd/system/sockets.target.wants/*initctl*; \
    rm -f /lib/systemd/system/basic.target.wants/*;\
    rm -f /lib/systemd/system/anaconda.target.wants/*;


RUN echo "Enabling kubelet ... " \
    && systemctl enable kubelet.service


RUN echo "Ensuring /etc/kubernetes/manifests" \
    && mkdir -p /etc/kubernetes/manifests
#
RUN echo "Adjusting systemd-tmpfiles timer" \
    && sed -i /usr/lib/systemd/system/systemd-tmpfiles-clean.timer -e 's#OnBootSec=.*#OnBootSec=1min#'


# squash
#FROM scratch
FROM ghcr.io/oracle/oraclelinux:8-slim
RUN  microdnf  update -y \
     && rm -rf /var/cache/yum/* \
     && microdnf clean all
COPY --from=build / /
#
# tell systemd that it is in docker (it will check for the container env)
# https://systemd.io/CONTAINER_INTERFACE/
ENV container docker
# systemd exits on SIGRTMIN+3, not SIGTERM (which re-executes it)
# https://bugzilla.redhat.com/show_bug.cgi?id=1201657
STOPSIGNAL SIGRTMIN+3

# NOTE: this is *only* for documentation, the entrypoint is overridden later
ENTRYPOINT [ "/usr/local/bin/entrypoint", "/sbin/init" ]