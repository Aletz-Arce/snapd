// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2018 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package builtin

const hostnameControlSummary = `allows configuring the system hostname`

const hostnameControlBaseDeclarationSlots = `
  hostname-control:
    allow-installation:
      slot-snap-type:
        - core
    deny-auto-connection: true
`

const hostnameControlConnectedPlugAppArmor = `
# Description: Can configure the system hostname.
# /{,usr/}bin/hostname ixr, # already allowed by default
/etc/hostname w,            # read allowed by default

#include <abstractions/dbus-strict>
/{,usr/}{,s}bin/hostnamectl           ixr,

# Allow access to hostname system service
dbus (send)
    bus=system
    path=/org/freedesktop/hostname1
    interface=org.freedesktop.DBus.Properties
    member="Get{,All}"
    peer=(label=unconfined),
dbus (receive)
    bus=system
    path=/org/freedesktop/hostname1
    interface=org.freedesktop.DBus.Properties
    member=PropertiesChanged
    peer=(label=unconfined),
dbus (send)
    bus=system
    path=/org/freedesktop/hostname1
    interface=org.freedesktop.DBus.Introspectable
    member=Introspect
    peer=(label=unconfined),
dbus(receive, send)
    bus=system
    path=/org/freedesktop/hostname1
    interface=org.freedesktop.hostname1
    member=Set{,Pretty,Static}Hostname
    peer=(label=unconfined),

# Needed to use 'sethostname'. See man 7 capabilities
capability sys_admin,
# Needed to use 'hostnamectl set-hostname'
capability sys_admin,
`

const hostnameControlConnectedPlugSecComp = `
# Description: Can configure the system hostname.
sethostname
`

func init() {
	registerIface(&commonInterface{
		name:                  "hostname-control",
		summary:               hostnameControlSummary,
		implicitOnCore:        true,
		implicitOnClassic:     true,
		baseDeclarationSlots:  hostnameControlBaseDeclarationSlots,
		connectedPlugAppArmor: hostnameControlConnectedPlugAppArmor,
		connectedPlugSecComp:  hostnameControlConnectedPlugSecComp,
		reservedForOS:         true,
	})

}
