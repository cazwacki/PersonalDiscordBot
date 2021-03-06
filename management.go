package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

/**
Used to validate a user's permissions before moving forward with a command. Prevents command abuse.
If the user has administrator permissions, just automatically allow them to perform any bot command.
**/
func userHasValidPermissions(s *discordgo.Session, m *discordgo.MessageCreate, permission int64) bool {
	perms, err := s.UserChannelPermissions(m.Author.ID, m.ChannelID)
	if err != nil {
		logError("Failed to acquire user permissions! " + err.Error())
		_, err = s.ChannelMessageSend(m.ChannelID, "Error occurred while validating your permissions.")
		if err != nil {
			logError("Failed to send error message! " + err.Error())
		}
		return false
	}
	if perms|permission == perms || perms|discordgo.PermissionAdministrator == perms {
		return true
	}
	return false
}

/**
Given a userID, generates a DM if one does not already exist with the user and sends the specified
message to them.
**/
func dmUser(s *discordgo.Session, userID string, message string) {
	channel, err := s.UserChannelCreate(userID)
	if err != nil {
		logError("Failed to create DM with user. " + err.Error())
		return
	}
	_, err = s.ChannelMessageSend(channel.ID, message)
	if err != nil {
		logError("Failed to send message! " + err.Error())
		return
	}
	logSuccess("Sent DM to user")
}

/**
A helper function for Handle_nick. Ensures the user targeted a user using @; if they did,
attempt to rename the specified user.
**/
func attemptRename(s *discordgo.Session, m *discordgo.MessageCreate, command []string) {
	logInfo(strings.Join(command, " "))
	regex := regexp.MustCompile(`^\<\@\!?[0-9]+\>$`)
	if regex.MatchString(command[1]) && len(command) > 2 {
		userID := stripUserID(command[1])
		err := s.GuildMemberNickname(m.GuildID, userID, strings.Join(command[2:], " "))
		if err == nil {
			_, err = s.ChannelMessageSend(m.ChannelID, "Done!")
			if err != nil {
				logError("Failed to send success message! " + err.Error())
				return
			}
			logSuccess("Successfully renamed user")
		} else {
			logError("Failed to set nickname! " + err.Error())
			_, err = s.ChannelMessageSend(m.ChannelID, err.Error())
			if err != nil {
				logError("Failed to send error message! " + err.Error())
			}
		}
		return
	}
	_, err := s.ChannelMessageSend(m.ChannelID, "Usage: `~nick @<user> <new name>`")
	if err != nil {
		logError("Failed to send usage message! " + err.Error())
	}
}

/**
A helper function for Handle_kick. Ensures the user targeted a user using @; if they did,
attempt to kick the specified user.
**/
func attemptKick(s *discordgo.Session, m *discordgo.MessageCreate, command []string) {
	logInfo(strings.Join(command, " "))
	regex := regexp.MustCompile(`^\<\@\!?[0-9]+\>$`)
	if len(command) >= 2 {
		if regex.MatchString(command[1]) {
			userID := stripUserID(command[1])
			if len(command) > 2 {
				reason := strings.Join(command[2:], " ")
				// dm user why they were kicked
				guild, err := s.Guild(m.GuildID)
				if err != nil {
					logError("Unable to load guild! " + err.Error())
				}
				guildName := "error: could not retrieve"
				if guild != nil {
					guildName = guild.Name
				}
				dmUser(s, userID, fmt.Sprintf("You have been kicked from **%s** by %s#%s because: %s\n", guildName, m.Author.Username, m.Author.Discriminator, reason))
				// kick with reason
				err = s.GuildMemberDeleteWithReason(m.GuildID, userID, reason)
				if err != nil {
					logError("Failed to kick user! " + err.Error())
					_, err = s.ChannelMessageSend(m.ChannelID, "Failed to kick the user.")
					if err != nil {
						logWarning("Failed to send failure message! " + err.Error())
					}
					return
				}
				_, err = s.ChannelMessageSend(m.ChannelID, ":wave: Kicked "+command[1]+" for the following reason: '"+reason+"'.")
				if err != nil {
					logWarning("Failed to send success message! " + err.Error())
					return
				}
				logSuccess("Kicked user with reason")
			} else {
				// dm user they were kicked
				guild, err := s.Guild(m.GuildID)
				if err != nil {
					logError("Unable to load guild! " + err.Error())
				}
				guildName := "error: could not retrieve"
				if guild != nil {
					guildName = guild.Name
				}
				dmUser(s, userID, fmt.Sprintf("You have been kicked from **%s** by %s#%s.\n", guildName, m.Author.Username, m.Author.Discriminator))
				// kick without reason
				err = s.GuildMemberDelete(m.GuildID, userID)
				if err != nil {
					logError("Failed to kick user! " + err.Error())
					_, err = s.ChannelMessageSend(m.ChannelID, "Failed to kick the user.")
					if err != nil {
						logWarning("Failed to send failure message! " + err.Error())
					}
					return
				}
				_, err = s.ChannelMessageSend(m.ChannelID, ":wave: Kicked "+command[1]+".")
				if err != nil {
					logWarning("Failed to send success message! " + err.Error())
					return
				}
				logSuccess("Kicked user")
			}
			return
		}
	}
	_, err := s.ChannelMessageSend(m.ChannelID, "Usage: `~kick @<user> (reason: optional)`")
	if err != nil {
		logError("Failed to send usage message! " + err.Error())
	}
}

/**
A helper function for Handle_ban. Ensures the user targeted a user using @; if they did,
attempt to ban the specified user.
**/
func attemptBan(s *discordgo.Session, m *discordgo.MessageCreate, command []string) {
	logInfo(strings.Join(command, " "))
	regex := regexp.MustCompile(`^\<\@\!?[0-9]+\>$`)
	if len(command) >= 2 {
		if regex.MatchString(command[1]) {
			userID := stripUserID(command[1])
			if len(command) > 2 {
				reason := strings.Join(command[2:], " ")
				// ban with reason
				err := s.GuildBanCreateWithReason(m.GuildID, userID, reason, 0)
				if err != nil {
					logError("Failed to ban user! " + err.Error())
					_, err = s.ChannelMessageSend(m.ChannelID, "Failed to ban the user.")
					if err != nil {
						logWarning("Failed to send failure message! " + err.Error())
					}
					return
				}
				// dm user why they were banned
				guild, err := s.Guild(m.GuildID)
				if err != nil {
					logError("Unable to load guild! " + err.Error())
				}
				guildName := "error: could not retrieve"
				if guild != nil {
					guildName = guild.Name
				}
				dmUser(s, userID, fmt.Sprintf("You have been banned from **%s** by %s#%s because: %s\n", guildName, m.Author.Username, m.Author.Discriminator, reason))

				_, err = s.ChannelMessageSend(m.ChannelID, ":hammer: Banned "+command[1]+" for the following reason: '"+reason+"'.")
				if err != nil {
					logWarning("Failed to send failure message! " + err.Error())
					return
				}
				logSuccess("Banned user with reason without issue")
			} else {
				// ban without reason
				err := s.GuildBanCreate(m.GuildID, userID, 0)
				if err != nil {
					logError("Failed to ban user! " + err.Error())
					_, err = s.ChannelMessageSend(m.ChannelID, "Failed to ban the user.")
					if err != nil {
						logWarning("Failed to send failure message! " + err.Error())
					}
					return
				}
				// dm user they were banned
				guild, err := s.Guild(m.GuildID)
				if err != nil {
					logError("Unable to load guild! " + err.Error())
				}
				guildName := "error: could not retrieve"
				if guild != nil {
					guildName = guild.Name
				}
				dmUser(s, userID, fmt.Sprintf("You have been banned from **%s** by %s#%s.\n", guildName, m.Author.Username, m.Author.Discriminator))
				_, err = s.ChannelMessageSend(m.ChannelID, ":hammer: Banned "+command[1]+".")
				if err != nil {
					logWarning("Failed to send failure message! " + err.Error())
					return
				}
				logSuccess("Banned user with reason without issue")
			}
			return
		}
	}
	_, err := s.ChannelMessageSend(m.ChannelID, "Usage: `~ban @<user> (reason: optional)`")
	if err != nil {
		logError("Failed to send failure message! " + err.Error())
	}
}

/**
Attempts to purge the last <number> messages, then removes the purge command.
*/
func attemptPurge(s *discordgo.Session, m *discordgo.MessageCreate, command []string) {
	logInfo(strings.Join(command, " "))
	if len(command) == 2 {
		messageCount, err := strconv.Atoi(command[1])
		if err != nil {
			_, err = s.ChannelMessageSend(m.ChannelID, "Usage: `~purge <number> (optional: @user)`")
			if err != nil {
				logError("Failed to send usage message! " + err.Error())
			}
			return
		}
		if messageCount < 1 {
			logWarning("User attempted to purge < 1 message.")
			_, err := s.ChannelMessageSend(m.ChannelID, ":frowning: Sorry, you must purge at least 1 message. Try again.")
			if err != nil {
				logError("Failed to send error message! " + err.Error())
			}
			return
		}
		for messageCount > 0 {
			messagesToPurge := 0
			// can only purge 100 messages per invocation
			if messageCount > 100 {
				messagesToPurge = 100
			} else {
				messagesToPurge = messageCount
			}

			// get the last (messagesToPurge) messages from the channel
			messages, err := s.ChannelMessages(m.ChannelID, messagesToPurge, m.ID, "", "")
			if err != nil {
				logError("Failed to pull messages from channel! " + err.Error())
				_, err = s.ChannelMessageSend(m.ChannelID, ":frowning: I couldn't pull messages from the channel. Try again.")
				if err != nil {
					logError("Failed to send error message! " + err.Error())
					return
				}
				return
			}

			// stop purging if there is nothing left to purge
			if len(messages) < messagesToPurge {
				messageCount = 0
			}

			// get the message IDs
			var messageIDs []string
			for _, message := range messages {
				messageIDs = append(messageIDs, message.ID)
			}

			// delete all the marked messages
			err = s.ChannelMessagesBulkDelete(m.ChannelID, messageIDs)
			if err != nil {
				logWarning("Failed to bulk delete messages! Attempting to continue... " + err.Error())
			}
			messageCount -= messagesToPurge
		}
		time.Sleep(time.Second)
		err = s.ChannelMessageDelete(m.ChannelID, m.ID)
		if err != nil {
			logError("Failed to delete invoked command! " + err.Error())
			return
		}
		logSuccess("Purged all messages, including command invoked")
	} else {
		_, err := s.ChannelMessageSend(m.ChannelID, "Usage: `~purge <number>`")
		if err != nil {
			logError("Failed to send usage message! " + err.Error())
		}
	}
}

/**
Attempts to copy over the last <number> messages to the given channel, then outputs its success
*/
func attemptCopy(s *discordgo.Session, m *discordgo.MessageCreate, command []string, preserveMessages bool) {
	logInfo(strings.Join(command, " "))
	var commandInvoked string
	if preserveMessages {
		commandInvoked = "cp"
	} else {
		commandInvoked = "mv"
	}
	if len(command) == 3 {
		messageCount, err := strconv.Atoi(command[1])
		if err != nil {
			_, err = s.ChannelMessageSend(m.ChannelID, "Usage: `~"+commandInvoked+" <number <= 100> <#channel>`")
			if err != nil {
				logError("Failed to send usage message! " + err.Error())
			}
			return
		}

		// verify correctly invoking channel
		if !strings.HasPrefix(command[2], "<#") || !strings.HasSuffix(command[2], ">") {
			_, err := s.ChannelMessageSend(m.ChannelID, "Usage: `~"+commandInvoked+" <number <= 100> <#channel>`")
			if err != nil {
				logError("Failed to send usage message! " + err.Error())
			}
			return
		}
		channel := strings.ReplaceAll(command[2], "<#", "")
		channel = strings.ReplaceAll(channel, ">", "")

		// retrieve messages from current invoked channel
		messages, err := s.ChannelMessages(m.ChannelID, messageCount, m.ID, "", "")
		if err != nil {
			_, err = s.ChannelMessageSend(m.ChannelID, "Ran into an error retrieving messages. :slight_frown:")
			if err != nil {
				logError("Failed to send error message! " + err.Error())
			}
			return
		}

		// construct an embed for each message
		for index := range messages {
			var embed discordgo.MessageEmbed
			embed.Type = "rich"
			message := messages[len(messages)-1-index]

			// remove messages if calling mv command
			if !preserveMessages {
				err := s.ChannelMessageDelete(m.ChannelID, message.ID)
				if err != nil {
					logWarning("Failed to delete a message. Attempting to continue... " + err.Error())
				}
			}

			// populating author information in the embed
			var embedAuthor discordgo.MessageEmbedAuthor
			if message.Author != nil {
				member, err := s.GuildMember(m.GuildID, message.Author.ID)
				nickname := ""
				if err == nil {
					nickname = member.Nick
				} else {
					logWarning("Could not find a nickname for the user! " + err.Error())
				}
				embedAuthor.Name = ""
				if nickname != "" {
					embedAuthor.Name += nickname + " ("
				}
				embedAuthor.Name += message.Author.Username + "#" + message.Author.Discriminator
				if nickname != "" {
					embedAuthor.Name += ")"
				}
				embedAuthor.IconURL = message.Author.AvatarURL("")
			}
			embed.Author = &embedAuthor

			// preserve message timestamp
			embed.Timestamp = string(message.Timestamp)
			var contents []*discordgo.MessageEmbedField

			// output message text
			logInfo("Message Content: " + message.Content)
			if message.Content != "" {
				embed.Description = message.Content
			}

			// output attachments
			logInfo(fmt.Sprintf("Attachments: %d\n", len(message.Attachments)))
			if len(message.Attachments) > 0 {
				for _, attachment := range message.Attachments {
					contents = append(contents, createField("Attachment: "+attachment.Filename, attachment.ProxyURL, false))
				}
			}

			// output embed contents (up to 10... jesus christ...)
			logInfo(fmt.Sprintf("Embeds: %d\n", len(message.Embeds)))
			if len(message.Embeds) > 0 {
				for _, embed := range message.Embeds {
					contents = append(contents, createField("Embed Title", embed.Title, false))
					contents = append(contents, createField("Embed Text", embed.Description, false))
					if embed.Image != nil {
						contents = append(contents, createField("Embed Image", embed.Image.ProxyURL, false))
					}
					if embed.Thumbnail != nil {
						contents = append(contents, createField("Embed Thumbnail", embed.Thumbnail.ProxyURL, false))
					}
					if embed.Video != nil {
						contents = append(contents, createField("Embed Video", embed.Video.URL, false))
					}
					if embed.Footer != nil {
						contents = append(contents, createField("Embed Footer", embed.Footer.Text, false))
					}
				}
			}

			// ouput reactions on a message
			if len(message.Reactions) > 0 {
				reactionText := ""
				for index, reactionSet := range message.Reactions {
					reactionText += reactionSet.Emoji.Name + " x" + strconv.Itoa(reactionSet.Count)
					if index < len(message.Reactions)-1 {
						reactionText += ", "
					}
				}
				contents = append(contents, createField("Reactions", reactionText, false))
			}
			embed.Fields = contents

			// send response
			_, err := s.ChannelMessageSendEmbed(channel, &embed)
			if err != nil {
				logError("Failed to send result message! " + err.Error())
				return
			}
		}
		_, err = s.ChannelMessageSend(m.ChannelID, "Copied "+strconv.Itoa(messageCount)+" messages from <#"+m.ChannelID+"> to <#"+channel+">! :smile:")
		if err != nil {
			logError("Failed to send success message! " + err.Error())
			return
		}
		logSuccess("Copied messages and sent success message")
	} else {
		_, err := s.ChannelMessageSend(m.ChannelID, "Usage: `~"+commandInvoked+" <number <= 100> <#channel>`")
		if err != nil {
			logError("Failed to send usage message! " + err.Error())
		}
	}
}

/**
Helper function for handleProfile. Attempts to retrieve a user's avatar and return it
in an embed.
*/
func attemptProfile(s *discordgo.Session, m *discordgo.MessageCreate, command []string) {
	logInfo(strings.Join(command, " "))
	if len(command) == 2 {
		regex := regexp.MustCompile(`^\<\@\!?[0-9]+\>$`)
		if regex.MatchString(command[1]) {
			userID := strings.TrimSuffix(command[1], ">")
			userID = strings.TrimPrefix(userID, "<@")
			userID = strings.TrimPrefix(userID, "!") // this means the user has a nickname
			var embed discordgo.MessageEmbed
			embed.Type = "rich"

			// get user
			user, err := s.User(userID)
			if err != nil {
				logError("Could not retrieve user from session! " + err.Error())
				_, err = s.ChannelMessageSend(m.ChannelID, "Error retrieving the user. :frowning:")
				if err != nil {
					logError("Failed to send error message! " + err.Error())
					return
				}
				return
			}

			// get member data from the user
			member, err := s.GuildMember(m.GuildID, userID)
			nickname := ""
			if err == nil {
				nickname = member.Nick
			} else {
				fmt.Println(err)
			}

			// title the embed
			embed.Title = "Profile Picture for "
			if nickname != "" {
				embed.Title += nickname + " ("
			}
			embed.Title += user.Username + "#" + user.Discriminator
			if nickname != "" {
				embed.Title += ")"
			}

			// attach the user's avatar as 512x512 image
			var image discordgo.MessageEmbedImage
			image.URL = user.AvatarURL("512")
			embed.Image = &image

			_, err = s.ChannelMessageSendEmbed(m.ChannelID, &embed)
			if err != nil {
				logError("Failed to send result message! " + err.Error())
				return
			}
			logSuccess("Returned user profile picture")
			return
		}
	}
	_, err := s.ChannelMessageSend(m.ChannelID, "Usage: `~profile @user`")
	if err != nil {
		logError("Failed to send usage message! " + err.Error())
		return
	}
}

func attemptAbout(s *discordgo.Session, m *discordgo.MessageCreate, command []string) {
	logInfo(strings.Join(command, " "))
	if len(command) == 2 {
		regex := regexp.MustCompile(`^\<\@\!?[0-9]+\>$`)
		if regex.MatchString(command[1]) {
			userID := stripUserID(command[1])

			logInfo(strings.Join(command, " "))

			member, err := s.GuildMember(m.GuildID, userID)
			if err != nil {
				logError("Could not retrieve user from the session! " + err.Error())
				_, err = s.ChannelMessageSend(m.ChannelID, "Error retrieving the user. :frowning:")
				if err != nil {
					logError("Failed to send error message! " + err.Error())
				}
				return
			}

			var embed discordgo.MessageEmbed
			embed.Type = "rich"

			// title the embed
			embed.Title = "About " + member.User.Username + "#" + member.User.Discriminator

			var contents []*discordgo.MessageEmbedField

			joinDate, err := member.JoinedAt.Parse()
			if err != nil {
				logError("Failed to parse Discord dates! " + err.Error())
				_, err := s.ChannelMessageSend(m.ChannelID, "Error parsing Discord's dates. :frowning:")
				if err != nil {
					logError("Failed to send error message! " + err.Error())
					return
				}
				return
			}

			nickname := "N/A"
			if member.Nick != "" {
				nickname = member.Nick
			}

			contents = append(contents, createField("Server Join Date", joinDate.Format("01/02/2006"), false))
			contents = append(contents, createField("Nickname", nickname, false))

			// get user's roles in readable form
			guildRoles, err := s.GuildRoles(m.GuildID)
			if err != nil {
				logError("Failed to retrieve guild roles! " + err.Error())
				_, err := s.ChannelMessageSend(m.ChannelID, "Error retrieving the guild's roles. :frowning:")
				if err != nil {
					logError("Failed to send error message! " + err.Error())
					return
				}
				return
			}
			var rolesAttached []string

			for _, role := range guildRoles {
				for _, roleID := range member.Roles {
					if role.ID == roleID {
						rolesAttached = append(rolesAttached, role.Name)
					}
				}
			}
			contents = append(contents, createField("Roles", strings.Join(rolesAttached, ", "), false))

			embed.Fields = contents

			// send response
			_, err = s.ChannelMessageSendEmbed(m.ChannelID, &embed)
			if err != nil {
				logError("Couldn't send the message... " + err.Error())
				return
			}
			logSuccess("Returned user information")
			return
		}
	}
	_, err := s.ChannelMessageSend(m.ChannelID, "Usage: `~about @user`")
	if err != nil {
		logError("Failed to send usage message! " + err.Error())
		return
	}
}

/**
Outputs the bot's current uptime.
**/
func handleUptime(s *discordgo.Session, m *discordgo.MessageCreate, start []string) {
	logInfo(start[0])
	start_time, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", start[0])
	if err != nil {
		logError("Could not parse start time! " + err.Error())
		_, err = s.ChannelMessageSend(m.ChannelID, "Error parsing the date... :frowning:")
		if err != nil {
			logError("Failed to send error message! " + err.Error())
			return
		}
	}
	_, err = s.ChannelMessageSend(m.ChannelID, ":robot: Uptime: "+time.Since(start_time).Truncate(time.Second/10).String())
	if err != nil {
		logError("Failed to send uptime message! " + err.Error())
		return
	}
	logSuccess("Reported uptime")
}

/**
Forces the bot to exit with code 0. Note that in Heroku the bot will restart automatically.
**/
func handleShutdown(s *discordgo.Session, m *discordgo.MessageCreate, command []string) {
	logInfo(strings.Join(command, " "))
	if m.Author.ID == "172311520045170688" {
		_, err := s.ChannelMessageSend(m.ChannelID, "Shutting Down.")
		if err != nil {
			logError("Failed to send shutdown message! " + err.Error())
		}
		s.Close()
		os.Exit(0)
	} else {
		_, err := s.ChannelMessageSend(m.ChannelID, "You dare try and go against the wishes of <@172311520045170688> ..? ")
		if err != nil {
			logError("Failed to send joke message! " + err.Error())
			return
		}
		time.Sleep(10 * time.Second)
		_, err = s.ChannelMessageSend(m.ChannelID, "Bruh this gonna be you when sage and his boys get here... I just pinged him so you better be afraid :slight_smile:")
		if err != nil {
			logError("Failed to send joke message! " + err.Error())
			return
		}
		time.Sleep(2 * time.Second)
		_, err = s.ChannelMessageSend(m.ChannelID, "https://media4.giphy.com/media/3o6Ztm3eJNDBy4NfiM/giphy.gif")
		if err != nil {
			logError("Failed to send joke message! " + err.Error())
			return
		}
	}
}

/**
Generates an invite code to the channel in which ~invite was invoked if the user has the
permission to create instant invites.
**/
func handleInvite(s *discordgo.Session, m *discordgo.MessageCreate, command []string) {
	logInfo(strings.Join(command, " "))
	if !userHasValidPermissions(s, m, discordgo.PermissionCreateInstantInvite) {
		logWarning("User attempted to create invite without proper permissions")
		_, err := s.ChannelMessageSend(m.ChannelID, "Sorry, you aren't allowed to create an instant invite.")
		if err != nil {
			logError("Failed to send permissions message! " + err.Error())
			return
		}
		return
	}
	var invite discordgo.Invite
	invite.Temporary = false
	invite.MaxAge = 21600 // 6 hours
	invite.MaxUses = 0    // infinite uses
	inviteResult, err := s.ChannelInviteCreate(m.ChannelID, invite)
	if err != nil {
		logError("Failed to generate invite! " + err.Error())
		_, err := s.ChannelMessageSend(m.ChannelID, "Error creating invite. Try again in a moment.")
		if err != nil {
			logError("Failed to send error message! " + err.Error())
		}
		return
	} else {
		_, err := s.ChannelMessageSend(m.ChannelID, ":mailbox_with_mail: Here's your invitation! https://discord.gg/"+inviteResult.Code)
		if err != nil {
			logError("Failed to send invite message! " + err.Error())
			return
		}
	}
	logSuccess("Generated and sent invite")
}

/**
Nicknames the user if they target themselves, or nicknames a target user if the user who invoked
~nick has the permission to change nicknames.
**/
func handleNickname(s *discordgo.Session, m *discordgo.MessageCreate, command []string) {
	if !(userHasValidPermissions(s, m, discordgo.PermissionChangeNickname) && strings.Contains(command[1], m.Author.ID)) && !(userHasValidPermissions(s, m, discordgo.PermissionManageNicknames)) {
		logWarning("User attempted to use nickname without proper permissions")
		_, err := s.ChannelMessageSend(m.ChannelID, "Sorry, you aren't allowed to change nicknames.")
		if err != nil {
			logError("Failed to send permissions message! " + err.Error())
		}
		return
	}
	attemptRename(s, m, command)
}

/**
Kicks a user from the server if the invoking user has the permission to kick users.
**/
func handleKick(s *discordgo.Session, m *discordgo.MessageCreate, command []string) {
	if !userHasValidPermissions(s, m, discordgo.PermissionKickMembers) {
		// validate caller has permission to kick other users
		logWarning("User attempted to use kick without proper permissions")
		_, err := s.ChannelMessageSend(m.ChannelID, "Sorry, you aren't allowed to kick users.")
		if err != nil {
			logError("Failed to send permissions message! " + err.Error())
		}
		return
	}
	attemptKick(s, m, command)
}

/**
Bans a user from the server if the invoking user has the permission to ban users.
**/
func handleBan(s *discordgo.Session, m *discordgo.MessageCreate, command []string) {
	if !userHasValidPermissions(s, m, discordgo.PermissionBanMembers) {
		// validate caller has permission to kick other users
		logWarning("User attempted to use ban without proper permissions")
		_, err := s.ChannelMessageSend(m.ChannelID, "Sorry, you aren't allowed to ban users.")
		if err != nil {
			logError("Failed to send permissions message! " + err.Error())
		}
		return
	}
	attemptBan(s, m, command)
}

/**
Removes the <number> most recent messages from the channel where the command was called.
**/
func handlePurge(s *discordgo.Session, m *discordgo.MessageCreate, command []string) {
	if !userHasValidPermissions(s, m, discordgo.PermissionManageMessages) {
		logWarning("User attempted to use purge without proper permissions")
		_, err := s.ChannelMessageSend(m.ChannelID, "Sorry, you aren't allowed to remove messages.")
		if err != nil {
			logError("Failed to send permissions message! " + err.Error())
		}
		return
	}
	attemptPurge(s, m, command)
}

/**
Copies the <number> most recent messages from the channel where the command was called and
pastes it in the requested channel.
**/
func handleCopy(s *discordgo.Session, m *discordgo.MessageCreate, command []string) {
	if !userHasValidPermissions(s, m, discordgo.PermissionManageMessages) {
		logWarning("User attempted to use copy without proper permissions")
		_, err := s.ChannelMessageSend(m.ChannelID, "Sorry, you aren't allowed to manage messages.")
		if err != nil {
			logError("Failed to send permissions message! " + err.Error())
		}
		return
	}
	attemptCopy(s, m, command, true)
}

/**
Same as above, but purges each message it copies
**/
func handleMove(s *discordgo.Session, m *discordgo.MessageCreate, command []string) {
	if !userHasValidPermissions(s, m, discordgo.PermissionManageMessages) {
		_, err := s.ChannelMessageSend(m.ChannelID, "Sorry, you aren't allowed to manage messages.")
		if err != nil {
			logError("Failed to send permissions message! " + err.Error())
		}
		return
	}
	attemptCopy(s, m, command, false)
}
