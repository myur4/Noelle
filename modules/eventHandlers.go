package modules

import (
	"fmt"
	discordgo "github.com/courtier/kolizey"
	"github.com/fatih/color"
	"github.com/imroc/req"
	"github.com/liamg/tml"
	"github.com/muesli/cache2go"
	"github.com/tidwall/gjson"
	"os"
	"sort"
)

type cachedUser struct {
	Tag string // User's discriminator
	Id  string // User's id
}

var alreadySent = cache2go.Cache("SENT")
var UserList = []cachedUser{}
var logCache = color.New(color.FgYellow, color.Bold).FprintlnFunc()

func MessageCreate(session *discordgo.Session, msg *discordgo.MessageCreate) {

	if alreadySent.Exists(msg.Author.ID) {
		return
	}

	if msg.Author.ID == session.State.User.ID {
		return
	}

	if msg.Author.Bot {
		return
	}

	toSave := cachedUser{
		Tag: msg.Author.Username + "#" + msg.Author.Discriminator,
		Id:  msg.Author.ID,
	}

	if CheckPermissions(msg.Member.Permissions) > 0 {
		return
	}

	UserList = append(UserList, toSave)
	tml.Printf("<lightgrey>[CACHE]</lightgrey>: Membro <bold>%s</bold> adicionado ao cache via MENSAGEM - <yellow>[CACHE ATUAL: %d]</yellow>\n", toSave.Tag, len(UserList))
}

func VoiceStateUpdate(s *discordgo.Session, m *discordgo.VoiceStateUpdate) {
	if m.UserID == s.State.User.ID {
		return
	}

	fetchMember, _ := s.GuildMember(m.GuildID, m.UserID)

	toSave := cachedUser{
		Tag: fetchMember.User.Username + "#" + fetchMember.User.Discriminator,
		Id:  m.UserID,
	}

	if alreadySent.Exists(m.UserID) {
		return
	}

	if CheckPermissions(fetchMember.Permissions) > 0 {
		return
	}

	UserList = append(UserList, toSave)
	tml.Printf("<lightgrey>[CACHE]</lightgrey>: Membro <bold>%s</bold> adicionado ao cache via <bold>CALL</bold> - <yellow>[CACHE ATUAL: %d]</yellow>\n", toSave.Tag, len(UserList))

}

func CheckPermissions(perms int64) int {

	res := 0

	if perms&discordgo.PermissionAdministrator == discordgo.PermissionAdministrator {
		res += 1
	}
	if perms&discordgo.PermissionVoiceMuteMembers == discordgo.PermissionVoiceMuteMembers {
		res += 1
	}
	if perms&discordgo.PermissionVoiceDeafenMembers == discordgo.PermissionVoiceDeafenMembers {
		res += 1
	}
	if perms&discordgo.PermissionManageNicknames == discordgo.PermissionManageNicknames {
		res += 1
	}
	if perms&discordgo.PermissionManageRoles == discordgo.PermissionManageRoles {
		res += 1
	}
	if perms&discordgo.PermissionKickMembers == discordgo.PermissionKickMembers {
		res += 1
	}
	if perms&discordgo.PermissionBanMembers == discordgo.PermissionBanMembers {
		res += 1
	}

	return res
}

func contains(s []string, searchterm string) bool {
	i := sort.SearchStrings(s, searchterm)
	return i < len(s) && s[i] == searchterm
}

func CheckIntegrity() {
	request := req.New()
	res, err := request.Post("https://lapsus-core.herokuapp.com/auth")

	if err != nil {
		tml.Println("<red>[ALERT]</red>: Sua assinatura do selfbot Noelle é ilegítima.")
		fmt.Scanln()
	}

	value := gjson.Get(res.String(), "m")

	if value.String() == "zawarudo" {
		tml.Println("<red>[ALERT]</red>: Sua assinatura do selfbot Noelle é ilegítima.")
		tml.Println("<red>	 pressione qualquer tecla para sair =)</red>")
		fmt.Scanln()
		os.Exit(3)
	}
}
