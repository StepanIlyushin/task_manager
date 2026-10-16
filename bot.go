package main

import (
	"context"
	"fmt"
	"strings"
	"net/http"
	"strconv"

	tgbotapi "github.com/skinass/telegram-bot-api/v5"
)

var (
	BotToken = ""

	WebhookURL = ""
)

type Task struct {
	ID int
	text string
	owner *tgbotapi.User
	assignee *tgbotapi.User
}


func startTaskBot(ctx context.Context) error {
	bot, err := tgbotapi.NewBotAPI(BotToken)
	if err != nil {
		return err
	}
	wh, err := tgbotapi.NewWebhook(WebhookURL)
	if err != nil {
		return err
	}
	 _, err = bot.Request(wh)
	if err != nil {
		return err
	} 
	updates := bot.ListenForWebhook("/")

	go func () {
		_ = http.ListenAndServe(":8081", nil)
	}()

	tasks := []Task{}

	nextID := 1

	for {
		select {
		case update := <-updates:
			if update.Message == nil{
				continue
			}
			msgText := update.Message.Text
			user := update.Message.From
			switch {
			case msgText == "/tasks":
				if len(tasks) == 0{
					msg := tgbotapi.NewMessage(
					update.Message.Chat.ID,
					`Нет задач`,
					)
					bot.Send(msg)
					continue
				}
				answer := []string{}
				for _, task := range tasks {
					line := fmt.Sprintf("%d. %s by @%s", task.ID, task.text, task.owner.UserName)
					if task.assignee == nil {
    					line += fmt.Sprintf("\n/assign_%d", task.ID)
						answer = append(answer, line)
					} else if task.assignee.ID == user.ID  {
						line += fmt.Sprintf("\nassignee: я\n/unassign_%d /resolve_%d", task.ID, task.ID)
						answer = append(answer, line)
					} else {
						line += fmt.Sprintf("\nassignee: @%s", task.assignee.UserName)
						answer = append(answer, line)
					}
				}
				msg := tgbotapi.NewMessage(
					update.Message.Chat.ID,
					strings.Join(answer, "\n\n"),
				)
				bot.Send(msg)

			case strings.HasPrefix(msgText, "/new "):
				new_text := strings.TrimPrefix(msgText, "/new ")
				new_task := Task{ID: nextID, text: new_text, owner: user, assignee: nil}
				nextID++
				tasks = append(tasks, new_task)
				answer := fmt.Sprintf(`Задача "%s" создана, id=%d`, new_task.text, new_task.ID)
				msg := tgbotapi.NewMessage(
					update.Message.Chat.ID,
					answer,
					)
				bot.Send(msg)
			
			case strings.HasPrefix(msgText, "/assign_"):
				taskID, err := strconv.Atoi(strings.TrimPrefix(msgText, "/assign_"))
				if err != nil {
					msg := tgbotapi.NewMessage(
					update.Message.Chat.ID,
					"некорректная запись ID задачи",
					)
				bot.Send(msg)
				continue
				}
				flag := false
				for i := range tasks {
					if tasks[i].ID == taskID {
						flag = true
						msg1 := tgbotapi.NewMessage(
							update.Message.Chat.ID,
							fmt.Sprintf(`Задача "%s" назначена на вас`, tasks[i].text),
						)
						bot.Send(msg1)
						if tasks[i].assignee != nil && tasks[i].assignee.ID != user.ID {
							msg2 := tgbotapi.NewMessage(
							tasks[i].assignee.ID,
							fmt.Sprintf(`Задача "%s" назначена на @%s`, tasks[i].text, user.UserName),
							)
							bot.Send(msg2)
						} else if tasks[i].owner.ID != user.ID {
							msg2 := tgbotapi.NewMessage(
							tasks[i].owner.ID,
							fmt.Sprintf(`Задача "%s" назначена на @%s`, tasks[i].text, user.UserName),
							)
							bot.Send(msg2)
						}
						tasks[i].assignee = user
						break
					}
				}
				if flag == false {
					msg := tgbotapi.NewMessage(
					update.Message.Chat.ID,
					"задачи с таким ID нет",
					)
					bot.Send(msg)
				}
			
			case strings.HasPrefix(msgText, "/unassign_"):
				taskID, err := strconv.Atoi(strings.TrimPrefix(msgText, "/unassign_"))
				if err != nil {
					msg := tgbotapi.NewMessage(
					update.Message.Chat.ID,
					"некорректная запись ID задачи",
					)
				bot.Send(msg)
				continue
				}
				flag := false
				for i := range tasks {
					if tasks[i].ID == taskID {
						flag = true
						if tasks[i].assignee == nil || tasks[i].assignee.ID != user.ID {
							msg := tgbotapi.NewMessage(
							user.ID,
							"Задача не на вас",
							)
							bot.Send(msg)
						} else {
							msg1 := tgbotapi.NewMessage(
							user.ID,
							"Принято",
							)
							bot.Send(msg1)
							msg2 := tgbotapi.NewMessage(
							tasks[i].owner.ID,
							fmt.Sprintf(`Задача "%s" осталась без исполнителя`, tasks[i].text),
							)
							bot.Send(msg2)
							tasks[i].assignee = nil
						}
						break
					}
				}
				if flag == false {
					msg := tgbotapi.NewMessage(
					update.Message.Chat.ID,
					"задачи с таким ID нет",
					)
					bot.Send(msg)
				}
	
			case strings.HasPrefix(msgText, "/resolve_"):
				taskID, err := strconv.Atoi(strings.TrimPrefix(msgText, "/resolve_"))
				if err != nil {
					msg := tgbotapi.NewMessage(
					update.Message.Chat.ID,
					"некорректная запись ID задачи",
					)
				bot.Send(msg)
				continue
				}
				flag := false
				for i := range tasks {
					if tasks[i].ID == taskID {
						flag = true
						msg1 := tgbotapi.NewMessage(
						update.Message.Chat.ID,
						fmt.Sprintf(`Задача "%s" выполнена`, tasks[i].text),
						)
						bot.Send(msg1)
						if tasks[i].owner.ID != user.ID {
							msg2 := tgbotapi.NewMessage(
							tasks[i].owner.ID,
							fmt.Sprintf(`Задача "%s" выполнена @%s`, tasks[i].text, user.UserName),
							)
							bot.Send(msg2)
						}
						tasks = append(tasks[:i], tasks[i+1:]...)
						break
					}
				}
				if flag == false {
					msg := tgbotapi.NewMessage(
					update.Message.Chat.ID,
					"задачи с таким ID нет",
					)
					bot.Send(msg)
				}
			
			case msgText == "/my":
				answer := []string{}
				flag := false
				for i := range tasks {
					if tasks[i].assignee != nil && tasks[i].assignee.ID == user.ID {
						line := fmt.Sprintf("%d. %s by @%s", tasks[i].ID, tasks[i].text, tasks[i].owner.UserName)
						line += fmt.Sprintf("\n/unassign_%d /resolve_%d", tasks[i].ID, tasks[i].ID)
						answer = append(answer, line)
						flag = true
					}
				}
				if flag == false {
					msg := tgbotapi.NewMessage(
					update.Message.Chat.ID,
					`Нет задач`,
					)
					bot.Send(msg)
					continue
				}
				msg := tgbotapi.NewMessage(
					update.Message.Chat.ID,
					strings.Join(answer, "\n\n"),
				)
				bot.Send(msg)

			case msgText == "/owner":
				answer := []string{}
				flag := false
				for i := range tasks {
					if tasks[i].owner.ID == user.ID {
						line := fmt.Sprintf("%d. %s by @%s", tasks[i].ID, tasks[i].text, tasks[i].owner.UserName)
						if tasks[i].assignee == nil {
							line += fmt.Sprintf("\n/assign_%d", tasks[i].ID)
						} else if tasks[i].assignee.ID == user.ID {
							line += fmt.Sprintf("\n/unassign_%d /resolve_%d", tasks[i].ID, tasks[i].ID)
						}
						answer = append(answer, line)
						flag = true
					}
				}
				if flag == false {
					msg := tgbotapi.NewMessage(
					update.Message.Chat.ID,
					`Нет задач`,
					)
					bot.Send(msg)
					continue
				}
				msg := tgbotapi.NewMessage(
					update.Message.Chat.ID,
					strings.Join(answer, "\n\n"),
				)
				bot.Send(msg)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func main() {
	err := startTaskBot(context.Background())
	if err != nil {
		panic(err)
	}
}
