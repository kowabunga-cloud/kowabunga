/*
 * Copyright (c) The Kowabunga Project
 * Apache License, Version 2.0 (see LICENSE or https://www.apache.org/licenses/LICENSE-2.0.txt)
 * SPDX-License-Identifier: Apache-2.0
 */

package kahuna

import (
	"fmt"

	"github.com/matcornic/hermes"
	gomail "gopkg.in/mail.v2"

	"github.com/kowabunga-cloud/common/klog"
)

const (
	EmailProductName      = "Kowabunga"
	EmailProductLink      = "htps://github.com/kowabunga-cloud/kowabunga"
	EmailProductLogoURL   = "https://raw.githubusercontent.com/kowabunga-cloud/infographics/master/art/kowabunga-title-white.png"
	EmailProductCopyright = "Copyright (c) The Kowabunga Project. All rights reserved."

	EmailCharacteristic = "Characteristic"
	EmailValue          = "Value"
)

func newHermes() hermes.Hermes {
	return hermes.Hermes{
		Theme: new(SmtpThemeKowabunga),
		Product: hermes.Product{
			Name:      EmailProductName,
			Link:      EmailProductLink,
			Logo:      EmailProductLogoURL,
			Copyright: EmailProductCopyright,
		},
	}
}

func sendEmail(to, subject string, body hermes.Email) error {
	h := newHermes()
	m := gomail.NewMessage()

	smtp := GetCfg().Global.SMTP
	m.SetHeader("From", smtp.From)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)

	// Generate the plaintext version of the e-mail (for clients that do not support xHTML)
	text, err := h.GeneratePlainText(body)
	if err != nil {
		return err
	}
	m.SetBody("text/plain", text)

	// Generate an HTML email with the provided contents (for modern clients)
	html, err := h.GenerateHTML(body)
	if err != nil {
		return err
	}
	m.AddAlternative("text/html", html)

	d := gomail.NewDialer(smtp.Host, smtp.Port, smtp.Username, smtp.Password)
	err = d.DialAndSend(m)
	if err != nil {
		klog.Error(err)
		return err
	}

	return nil
}

func quotaToString(val int, size bool) string {
	if val != 0 {
		if size {
			return HumanByteSize(uint64(val))
		}
		return fmt.Sprintf("%d", val)
	}
	return "Unlimited"
}

func NewEmailProjectCreated(prj *Project, user *User) error {
	subject := fmt.Sprintf("Your new project %s has been created", prj.Name)

	instances := quotaToString(int(prj.Quotas.InstancesCount), false)
	vcpus := quotaToString(int(prj.Quotas.VCPUs), false)
	mem := quotaToString(int(prj.Quotas.MemorySize), true)
	disk := quotaToString(int(prj.Quotas.StorageSize), true)

	email := hermes.Email{
		Body: hermes.Body{
			Greeting: fmt.Sprintf("Hi %s", user.Name),
			Intros: []string{
				fmt.Sprintf("Congratulations, your new project %s has been successfully created ! hope you're gonna make a good use of it.", prj.Name),
			},
			Table: hermes.Table{
				Data: [][]hermes.Entry{
					{
						{Key: EmailCharacteristic, Value: "Kahuna Controller"},
						{Key: EmailValue, Value: GetCfg().Global.PublicURL},
					},
					{
						{Key: EmailCharacteristic, Value: "Name"},
						{Key: EmailValue, Value: prj.Name},
					},
					{
						{Key: EmailCharacteristic, Value: "Domain"},
						{Key: EmailValue, Value: prj.Domain},
					},
					{
						{Key: EmailCharacteristic, Value: "Instances Limit"},
						{Key: EmailValue, Value: instances},
					},
					{
						{Key: EmailCharacteristic, Value: "vCPUs Limit"},
						{Key: EmailValue, Value: vcpus},
					},
					{
						{Key: EmailCharacteristic, Value: "Memory Limit"},
						{Key: EmailValue, Value: mem},
					},
					{
						{Key: EmailCharacteristic, Value: "Storage Limit"},
						{Key: EmailValue, Value: disk},
					},
				},
			},
		},
	}

	for zone, subnetId := range prj.PrivateSubnets {
		s, err := FindSubnetByID(subnetId)
		if err != nil {
			continue
		}

		sub := fmt.Sprintf("%s VPC Subnet", zone)
		gw := fmt.Sprintf("%s VPC Gateway", zone)
		dns := fmt.Sprintf("%s VPC DNS", zone)
		entry := [][]hermes.Entry{
			{
				{Key: EmailCharacteristic, Value: sub},
				{Key: EmailValue, Value: s.CIDR},
			},
			{
				{Key: EmailCharacteristic, Value: gw},
				{Key: EmailValue, Value: s.Gateway},
			},
			{
				{Key: EmailCharacteristic, Value: dns},
				{Key: EmailValue, Value: s.DNS},
			},
		}
		email.Body.Table.Data = append(email.Body.Table.Data, entry...)
	}

	return sendEmail(user.Email, subject, email)
}

func NewEmailInstanceCreated(instance *Instance, user *User) error {
	prj, err := instance.Project()
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("Your new instance %s has been created", instance.Name)

	vol, err := FindVolumeByID(instance.Disks[fmt.Sprintf("%sa", VolumeOsDiskPrefix)])
	if err != nil {
		return err
	}
	t, err := vol.Template()
	if err != nil {
		return err
	}
	os := fmt.Sprintf("%s (%s)", t.Name, t.OS)

	email := hermes.Email{
		Body: hermes.Body{
			Greeting: fmt.Sprintf("Hi %s", user.Name),
			Intros: []string{
				"Congratulations, your new instance has been successfully created ! hope you're gonna make a good use of it.",
			},
			Table: hermes.Table{
				Data: [][]hermes.Entry{
					{
						{Key: EmailCharacteristic, Value: "Kahuna Controller"},
						{Key: EmailValue, Value: GetCfg().Global.PublicURL},
					},
					{
						{Key: EmailCharacteristic, Value: "Hostname"},
						{Key: EmailValue, Value: instance.Name},
					},
					{
						{Key: EmailCharacteristic, Value: "Domain"},
						{Key: EmailValue, Value: prj.Domain},
					},
					{
						{Key: EmailCharacteristic, Value: "OS"},
						{Key: EmailValue, Value: os},
					},
					{
						{Key: EmailCharacteristic, Value: "vCPUs"},
						{Key: EmailValue, Value: fmt.Sprintf("%d", instance.CPU)},
					},
					{
						{Key: EmailCharacteristic, Value: "Memory"},
						{Key: EmailValue, Value: HumanByteSize(uint64(instance.Memory))},
					},
					{
						{Key: EmailCharacteristic, Value: "NICs"},
						{Key: EmailValue, Value: fmt.Sprintf("%d", len(instance.Interfaces))},
					},
					{
						{Key: EmailCharacteristic, Value: "Disks"},
						{Key: EmailValue, Value: fmt.Sprintf("%d", len(instance.Disks))},
					},
					{
						{Key: EmailCharacteristic, Value: "Service Account"},
						{Key: EmailValue, Value: prj.BootstrapUser},
					},
					{
						{Key: EmailCharacteristic, Value: "Root Password"},
						{Key: EmailValue, Value: instance.RootPassword},
					},
					{
						{Key: EmailCharacteristic, Value: "Estimated Monthly Cost"},
						{Key: EmailValue, Value: fmt.Sprintf("%.2f EUR", instance.Cost.Price)},
					},
				},
			},
		},
	}

	publicIP := instance.GetIpAddress(false)
	if publicIP != "" {
		val := "Public Interface Address"
		entry := []hermes.Entry{
			{Key: EmailCharacteristic, Value: val},
			{Key: EmailValue, Value: publicIP},
		}
		email.Body.Table.Data = append(email.Body.Table.Data, entry)
	}

	privateIP := instance.GetIpAddress(true)
	val := "Private Interface Address"
	entry := []hermes.Entry{
		{Key: EmailCharacteristic, Value: val},
		{Key: EmailValue, Value: privateIP},
	}
	email.Body.Table.Data = append(email.Body.Table.Data, entry)

	return sendEmail(user.Email, subject, email)
}

func NewEmailUserCreated(user *User) error {
	subject := "Welcome to Kowabunga !"

	confirmationUrl := fmt.Sprintf("%s/confirm?user=%s&token=%s", GetCfg().Global.PublicURL, user.String(), user.RegistrationToken)
	notify := "Disabled"
	if user.NotificationsEnabled {
		notify = "Enabled"
	}

	email := hermes.Email{
		Body: hermes.Body{
			Name: user.Name,
			Intros: []string{
				"Welcome to Kowabunga ! We're very excited to have you on board.",
			},
			Dictionary: []hermes.Entry{
				{Key: "Name", Value: user.Name},
				{Key: "Role", Value: user.Role},
				{Key: "Email Notifications", Value: notify},
			},
			Actions: []hermes.Action{
				{
					Instructions: "To get started with Kowabunga, please click here:",
					Button: hermes.Button{
						Text: "Confirm your account",
						Link: confirmationUrl,
					},
				},
			},
			Outros: []string{
				"Need help, or have questions? Just reply to this email, we'd love to help.",
			},
		},
	}

	return sendEmail(user.Email, subject, email)
}

func NewEmailUserPasswordConfirmation(user *User) error {
	subject := "Forgot about your Kowabunga password ?"

	confirmationUrl := fmt.Sprintf("%s/confirmForgotPassword?user=%s&token=%s", GetCfg().Global.PublicURL, user.String(), user.PasswordRenewalToken)
	email := hermes.Email{
		Body: hermes.Body{
			Name: user.Name,
			Intros: []string{
				"Welcome back to Kowabunga ! It seems you've forgotten your password, that happens.",
			},
			Actions: []hermes.Action{
				{
					Instructions: "To confirm password reset, please click here:",
					Button: hermes.Button{
						Text: "Confirm password renewal",
						Link: confirmationUrl,
					},
				},
			},
			Outros: []string{
				"Need help, or have questions? Just reply to this email, we'd love to help.",
			},
		},
	}

	return sendEmail(user.Email, subject, email)
}

func NewEmailUserPassword(user *User, password string) error {
	subject := "Your Kowabunga password has been reset !"

	email := hermes.Email{
		Body: hermes.Body{
			Name: user.Name,
			Intros: []string{
				"Welcome back to Kowabunga ! It seems you've forget your password, that happens.",
			},
			Actions: []hermes.Action{
				{
					Instructions: "Here's your new password:",
					InviteCode:   password,
				},
			},
			Outros: []string{
				"This email is a one-timer. We won't be able to recover your password if lost, a new one will have to be generated again",
			},
		},
	}

	return sendEmail(user.Email, subject, email)
}

func NewEmailAgentApiToken(agent *Agent, apikey string) error {
	subject := "A new Kowabunga agent API key has been set !"

	email := hermes.Email{
		Body: hermes.Body{
			Name: agent.Name,
			Intros: []string{
				fmt.Sprintf("A new server-to-server API key has been requested for %s agent %s (%s).", agent.Type, agent.Name, agent.String()),
			},
			Actions: []hermes.Action{
				{
					Instructions: "Here's agent's new API key:",
					InviteCode:   apikey,
				},
			},
			Outros: []string{
				"This email is a one-timer. We won't be able to recover your API key if lost, a new one will have to be generated again",
			},
		},
	}

	return sendEmail(GetCfg().Global.AdminEmail, subject, email)
}

func NewEmailUserApiToken(user *User, apikey string) error {
	subject := "A new Kowabunga API key has been set !"

	email := hermes.Email{
		Body: hermes.Body{
			Name: user.Name,
			Intros: []string{
				"A new server-to-server API key has been requested.",
			},
			Actions: []hermes.Action{
				{
					Instructions: "Here's your new API key:",
					InviteCode:   apikey,
				},
			},
			Outros: []string{
				"This email is a one-timer. We won't be able to recover your API key if lost, a new one will have to be generated again",
			},
		},
	}

	return sendEmail(user.Email, subject, email)
}
