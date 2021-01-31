package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/md4"
)

var groups []Group // Массив для всех груп
var tasks []Task   // Массив для всех задач

// Group ...
type Group struct {
	Name        string `json:"group_name"`
	Description string `json:"group_description"`
	ID          int    `json:"group_id"`
	Parent      int    `json:"parent_id"`
}

// Task ...
type Task struct {
	ID          string `json:"task_id"`
	Group       int    `json:"group_id"`
	Task        string `json:"task"`
	Completed   bool   `json:"completed"`
	CreatedAt   string `json:"Created At"`
	CompletedAt string `json:"Completed At"`
}

func getGroups(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)

	groupsCopy := make([]Group, len(groups))
	copy(groupsCopy, groups)
	var response []Group // Слайс, который высылается в ответ

	filter := c.QueryParam("sort")

	if filter != "" {

		// sort = "parent_with_childs"
		if filter == "parent_with_childs" {
			for _, parent := range groupsCopy {
				id := parent.ID
				response = append(response, parent)
				for _, child := range groupsCopy {
					if child.Parent == id {
						response = append(response, child)
					}
				}
			}
		}

		// сразу сортируем по имени
		sort.Slice(groupsCopy, func(i, j int) bool { return groupsCopy[i].ID < groupsCopy[j].ID })

		// sort = parents_first
		if filter == "parents_first" {
			var parents []Group // создаем слайс с родителями
			var childs []Group  // и с детьми
			for _, group := range groupsCopy {
				if group.Parent == 0 {
					parents = append(parents, group)
				} else {
					childs = append(childs, group)
				}
			}
			// Конкатенируем слайсы
			response = append(response, parents...)
			response = append(response, childs...)
		}

		if filter == "name" {
			response = append(response, groupsCopy...)
		}

	}

	// Проверяем выставлен ли парметр лимита и обрезаем слайс
	limit := c.QueryParam("limit")
	if limit != "0" && limit != "" {
		lim, _ := strconv.Atoi(limit)
		response = response[:lim]
	}

	c.Response().WriteHeader(http.StatusOK)
	return json.NewEncoder(c.Response()).Encode(response)
}

func getTopGroups(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)

	// Если группа не имеет родителей, то она попадает в массив response
	var response []Group
	for _, group := range groups {
		if group.Parent == 0 {
			response = append(response, group)
		}
	}

	c.Response().WriteHeader(http.StatusOK)
	return json.NewEncoder(c.Response()).Encode(response)
}

func getGroupInfo(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	for _, g := range groups {
		if g.ID == id {
			return c.JSON(http.StatusOK, g)
		}
	}

	return c.NoContent(http.StatusNoContent) // Если группа id не найдена - отправляем ошибку 204
}

func getGroupChilds(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "you didn't provide parameters!") // Возвращаем ошибку 400, если параметр не передан
	}

	var response []Group
	exist := false // для проверки существования группу с данным id
	for _, g := range groups {
		if g.ID == id { // Если группа существует
			exist = true // меняем флаг
		}
		if g.Parent == id {
			response = append(response, g)
		}
	}

	// Если группы с данным id не существует - возвращаем статус 204
	if !exist {
		return c.NoContent(http.StatusNoContent)
	}

	c.Response().WriteHeader(http.StatusOK)
	return json.NewEncoder(c.Response()).Encode(response)
}

func createNewGroup(c echo.Context) error {
	group := new(Group)
	if err := c.Bind(group); err != nil {
		return c.String(http.StatusConflict, "Something went wrong") // Статус 409
	}
	// Имя группы - обязательное поле
	if group.Name == "" {
		return &echo.HTTPError{Code: http.StatusBadRequest, Message: "Required field is missing: group_name"}
	}

	if group.ID == 0 {
		group.ID = len(groups) + 1
	}

	groups = append(groups, *group)

	return c.JSON(http.StatusCreated, group)
}

func updateGroup(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "you didn't provide parameters!") // Возвращаем ошибку 400, если параметр не передан
	}

	// Создаем новую сущность группы с обновленными параметрами
	var group Group
	if err := c.Bind(group); err != nil {
		return c.String(http.StatusConflict, "Something went wrong") // Статус 409
	}
	if group.Name == "" {
		return &echo.HTTPError{Code: http.StatusBadRequest, Message: "Required field is missing: group_name"}
	}
	// id обязательно должно быть тем же самым, которое представленно в параметрах запроса
	group.ID = id

	for i, g := range groups {
		if g.ID == id {
			groups[i] = group // заменяем старую группу новой
		}
	}

	return c.JSON(http.StatusOK, group)
}

func deleteGroup(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Wrong id format")
	}

	groupsCopy := make([]Group, len(groups))
	copy(groupsCopy, groups)
	sort.Slice(groupsCopy, func(i, j int) bool { return groupsCopy[i].ID < groupsCopy[j].ID })

	for _, g := range groups {
		if g.ID == id {
			// Проверяем есть ли у группы дочерние группы
			for _, g := range groups {
				if g.Parent == id {
					return c.String(http.StatusBadRequest, "Can't delete this group")
				}
			}
			for _, t := range tasks {
				if t.Group == id {
					return c.String(http.StatusBadRequest, "Can't delete this group")
				}
			}
			return c.String(http.StatusOK, "Successfuly deleted")
		}
	}

	return c.String(http.StatusBadRequest, "Group doesn't exist")
}

func getTasks(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)

	filter := c.QueryParam("sort")
	typeOfTask := c.QueryParam("type")
	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil {
		limit = 0 // если параметр с лимитом не передали - присваиваем ему значение 0
	}

	var response []Task
	// сначала сортируем по типу, закидывая все подходящие элементы в массив response
	// чтобы в дальнейшем не пришлось сортировать лишнее по имени
	switch typeOfTask {
	case "all":
		copy(response, tasks)
		break
	case "completed":
		for _, task := range tasks {
			if task.Completed == true {
				response = append(response, task)
			}
		}
		break
	case "working":
		for _, task := range tasks {
			if task.Completed == false {
				response = append(response, task)
			}
		}
		break
	default:
		return c.String(http.StatusBadRequest, "Invalid type parameter")
	}

	// Сортировка по параметру sort
	switch filter {
	case "name":
		sort.Slice(response, func(i, j int) bool { return response[i].Task < response[j].Task })
		break
	case "group":
		sort.Slice(response, func(i, j int) bool { return response[i].Group < response[j].Group })
		break
	default:
		return c.String(http.StatusBadRequest, "Invalid sort parameter")
	}

	c.Response().WriteHeader(http.StatusOK)
	if limit != 0 {
		return json.NewEncoder(c.Response()).Encode(response[:limit])
	}

	return json.NewEncoder(c.Response()).Encode(response)
}

func createNewTask(c echo.Context) error {
	task := new(Task)
	if err := c.Bind(task); err != nil {
		return c.String(http.StatusConflict, "Something went wrong, check ur request data") // Статус 409
	}

	if task.Group == 0 || task.Task == "" {
		return c.String(http.StatusBadRequest, "Required fields is missing")
	}

	// Проверяем, существует ли группа указанная в запросе
	for _, g := range groups {
		if g.ID == task.Group {
			return c.String(http.StatusBadRequest, "The group specified in the request does not exist")
		}
	}

	// Хэшируем текст задачи и ее группу алгоритмом md4, так как нам нужны первые 5 символов
	h := md4.New()
	h.Write([]byte(task.Task + strconv.Itoa(task.Group)))
	// Присваиваем task_id первые 5 символов хэша
	task.ID = string(h.Sum(nil)[:5])
	// Проверяем, есть ли задание с таким же хэшем
	for _, t := range tasks {
		if t.ID == task.ID {
			return c.String(http.StatusBadRequest, "This Task already exist")
		}
	}

	task.CreatedAt = time.Now().String()
	task.CompletedAt = ""
	task.Completed = false

	tasks = append(tasks, *task)
	return c.JSON(http.StatusAccepted, task)
}

func updateTask(c echo.Context) error {
	taskID := c.Param("id")

	task := new(Task)
	if err := c.Bind(task); err != nil {
		return c.String(http.StatusConflict, "Something went wrong, check ur request data") // Статус 409
	}

	if task.Group == 0 || task.Task == "" {
		return c.String(http.StatusBadRequest, "Required fields is missing")
	}

	// Хэшируем текст задачи и ее группу алгоритмом md4, так как нам нужны первые 5 символов
	h := md4.New()
	h.Write([]byte(task.Task + strconv.Itoa(task.Group)))
	// Присваиваем task_id первые 5 символов хэша
	task.ID = string(h.Sum(nil)[:5])
	// Проверяем, есть ли задание с таким же хэшем
	for _, t := range tasks {
		if t.ID == task.ID {
			return c.String(http.StatusBadRequest, "This Task already exist")
		}
	}

	for _, t := range tasks {
		if t.ID == taskID {
			t.ID = task.ID
			t.Task = task.Task
			t.Group = task.Group
			task = &t
		}
	}

	return c.JSON(http.StatusAccepted, task)

}

func tasksByGroup(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Wrong id format")
	}
	typeOfTask := c.QueryParam("type")

	var response []Task
	for _, t := range tasks {
		if t.Group == id {
			switch typeOfTask {
			case "all":
				response = append(response, t)
				break
			case "completed":
				if t.Completed == true {
					response = append(response, t)
				}
				break
			case "working":
				if t.Completed == false {
					response = append(response, t)
				}
				break
			default:
				return c.String(http.StatusBadRequest, "Type is invalid")
			}
		}
	}

	return c.JSON(http.StatusOK, response)
}

func taskStatusTurn(c echo.Context) error {
	id := c.Param("id")

	finished := c.QueryParam("finished")
	if finished != "true" && finished != "false" {
		return c.String(http.StatusBadRequest, "Parameter is invalid")
	}

	for _, g := range tasks {
		if g.ID == id {
			if strconv.FormatBool(g.Completed) == finished {
				return c.String(http.StatusOK, "The status of the task has already matches your request")
			}
			g.Completed = !g.Completed

			// Меняем поле Completed At если задание считается выполненным
			if finished == "true" {
				g.CompletedAt = time.Now().String()
				// И на пустую строку если не выполненно
			} else {
				g.CompletedAt = ""
			}

			return c.JSON(http.StatusOK, g)
		}
	}

	return c.String(http.StatusBadRequest, "This task does not exist")
}

func main() {
	router := echo.New()

	groups = append(groups,
		Group{Name: "One", ID: 1, Parent: 2},
		Group{Name: "Two", ID: 2, Parent: 0},
		Group{Name: "Three", ID: 3, Parent: 2},
		Group{Name: "Four", ID: 4, Parent: 1},
	)

	router.GET("/groups", getGroups)
	router.GET("/group/top_parents", getTopGroups)
	router.GET("/group/:id", getGroupInfo)
	router.GET("/group/childs/:id", getGroupChilds)
	router.POST("/group/new", createNewGroup)
	router.PUT("/group/:id", updateGroup)
	router.DELETE("/group/:id", deleteGroup)

	router.GET("/tasks", getTasks)
	router.POST("/tasks/new", createNewTask)
	router.PUT("/tasks/:id", updateTask)
	router.GET("tasks/group/:id", tasksByGroup)
	router.POST("/tasks/:id", taskStatusTurn)

	router.Logger.Fatal(router.Start(":1323"))
}
