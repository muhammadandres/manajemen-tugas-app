package repository

import (
	"errors"
	"fmt"
	"log"
	"manajemen_tugas_master/model/domain"

	"gorm.io/gorm"
)

type taskAndOwnerRepository struct {
	db *gorm.DB
}

func NewTaskAndOwnerRepository(db *gorm.DB) TaskAndOwnerRepository {
	return &taskAndOwnerRepository{db}
}

func (t *taskAndOwnerRepository) Create(user *domain.User, task *domain.Task) (*domain.Task, *domain.Owner, error) {
	var Owner domain.Owner
	Owner.Email = user.Email
	Owner.UserID = user.ID
	if err := t.db.Create(&Owner).Error; err != nil {
		return nil, nil, err
	}

	task.OwnerID = Owner.ID
	if err := t.db.Create(&task).Error; err != nil {
		return nil, nil, err
	}

	return task, &Owner, nil
}

func (t *taskAndOwnerRepository) FindById(id uint) (*domain.Task, error) {
	var task domain.Task
	var owner domain.Owner

	// Mencari data owner
	if err := t.db.Find(&owner).Error; err != nil {
		return nil, errors.New("Owner not found")
	}
	// Mencari semua data task tertentu dengan semua relasinya
	if err := t.db.Preload("PlanningFile").Preload("ProjectFile").Preload("Manager").Preload("Employee").First(&task, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("Task not found")
		}
		return nil, err
	}

	// Mengisi informasi owner pada task
	task.Owner = owner

	return &task, nil
}

func (t *taskAndOwnerRepository) FindAll() ([]*domain.Task, error) {
	var tasks []*domain.Task
	var owners []*domain.Owner

	// Mencari semua data owner
	if err := t.db.Find(&owners).Error; err != nil {
		return nil, errors.New("Owner not found")
	}

	// Mencari semua data task dengan semua relasinya
	if err := t.db.Preload("Manager").Preload("Employee").Preload("PlanningFile").Preload("ProjectFile").Find(&tasks).Error; err != nil {
		return nil, errors.New("Task not found")
	}

	// Mengisi informasi owner pada setiap task
	for _, task := range tasks {
		for _, owner := range owners {
			if owner.ID == task.OwnerID {
				task.Owner = *owner
				break
			}
		}
	}

	return tasks, nil
}

func (t *taskAndOwnerRepository) FindAllOwners() ([]*domain.Task, error) {
	var tasks []*domain.Task
	var owners []*domain.Owner

	// Mencari semua data owner
	if err := t.db.Find(&owners).Error; err != nil {
		return nil, errors.New("Owner not found")
	}

	// Mencari semua data task
	if err := t.db.Find(&tasks).Error; err != nil {
		return nil, errors.New("Owner not found")
	}

	for _, task := range tasks {
		for _, owner := range owners {
			task.Owner = *owner
		}
	}

	return tasks, nil
}

func (t *taskAndOwnerRepository) FindAllManagers() ([]*domain.Task, error) {
	var tasks []*domain.Task

	// Mencari semua data task dengan preload untuk Manager
	if err := t.db.Preload("Manager").Find(&tasks).Error; err != nil {
		return nil, errors.New("Failed to find tasks")
	}

	return tasks, nil
}

func (t *taskAndOwnerRepository) FindAllEmployees() ([]*domain.Task, error) {
	var tasks []*domain.Task

	// Mencari semua data task dengan preload untuk Manager
	if err := t.db.Preload("Employee").Find(&tasks).Error; err != nil {
		return nil, errors.New("Failed to find tasks")
	}

	return tasks, nil
}

func (t *taskAndOwnerRepository) FindAllPlanningFiles() ([]*domain.Task, error) {
	var tasks []*domain.Task

	// Mencari semua data task dengan preload untuk Manager
	if err := t.db.Preload("PlanningFile").Find(&tasks).Error; err != nil {
		return nil, errors.New("Failed to find tasks")
	}

	return tasks, nil
}

func (t *taskAndOwnerRepository) FindAllProjectFiles() ([]*domain.Task, error) {
	var tasks []*domain.Task

	// Mencari semua data task dengan preload untuk Manager
	if err := t.db.Preload("ProjectFile").Find(&tasks).Error; err != nil {
		return nil, errors.New("Failed to find tasks")
	}

	return tasks, nil
}

func (t *taskAndOwnerRepository) Update(task *domain.Task, manager *domain.Manager, employee *domain.Employee, planningFile *domain.PlanningFile, projectFile *domain.ProjectFile) (*domain.Task, *domain.Manager, *domain.Employee, *domain.PlanningFile, *domain.ProjectFile, error) {
	// Simpan task
	if task.NameTask != "" || task.PlanningDescription != "" || task.PlanningStatus != "" || task.ProjectStatus != "" || task.PlanningDueDate != "" || task.ProjectDueDate != "" || task.Priority != "" || task.ProjectComment != "" {
		if err := t.db.Save(&task).Error; err != nil {
			return nil, nil, nil, nil, nil, err
		}
	}

	// Simpan manager
	if manager != nil && (manager.Email != "") {
		var user domain.User
		if err := t.db.First(&user, "email = ?", manager.Email).Error; err != nil {
			return nil, nil, nil, nil, nil, errors.New("User not found")
		}

		// validasi agar ada tidak ada user yang sama pada manager
		var countManager int64
		err := t.db.Model(&domain.Manager{}).
			Where("user_id = ?", user.ID).                                         // Filter by user_id
			Joins("JOIN task_managers ON task_managers.manager_id = managers.id"). // Join with task_managers
			Where("task_managers.task_id = ?", task.ID).                           // Filter by task_id
			Count(&countManager).Error
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		if countManager > 0 {
			return nil, nil, nil, nil, nil, errors.New("User is already assigned as manager to a task")
		}

		// validasi agar user yang telah menjadi employee tidak bisa menjadi manager lagi pada task yang sama.
		var countEmployee int64
		err = t.db.Model(&domain.Employee{}).
			Where("user_id = ?", user.ID).                                             // Filter by user_id
			Joins("JOIN task_employees ON task_employees.employee_id = employees.id"). // Join with task_employees
			Where("task_employees.task_id = ?", task.ID).                              // Filter by task_id
			Count(&countEmployee).Error
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		if countEmployee > 0 {
			return nil, nil, nil, nil, nil, errors.New("User is already assigned as employee to a task")
		} else {
			// jika kedua validasi tersebut berhasil masukkan data ke table penghubung
			manager.UserID = user.ID
			if err := t.db.Save(manager).Error; err != nil {
				return nil, nil, nil, nil, nil, errors.New("Failed to save manager data")
			}
			sqlQuery := "INSERT INTO task_managers (task_id, manager_id) VALUES (?, ?)"
			if err := t.db.Exec(sqlQuery, task.ID, manager.ID).Error; err != nil {
				return nil, nil, nil, nil, nil, err
			}
		}
	}

	// Simpan employee
	if employee != nil && (employee.Email != "") {
		var user domain.User
		if err := t.db.First(&user, "email = ?", employee.Email).Error; err != nil {
			return nil, nil, nil, nil, nil, errors.New("User not found")
		}

		// validasi agar ada tidak ada user yang sama pada employee
		var countEmployee int64
		err := t.db.Model(&domain.Employee{}).
			Where("user_id = ?", user.ID).                                             // Filter by user_id
			Joins("JOIN task_employees ON task_employees.employee_id = employees.id"). // Join with task_employees
			Where("task_employees.task_id = ?", task.ID).                              // Filter by task_id
			Count(&countEmployee).Error
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		if countEmployee > 0 {
			return nil, nil, nil, nil, nil, errors.New("User is already assigned as employee to a task")
		}

		// validasi agar user yang telah menjadi manager tidak bisa menjadi employee lagi pada task yang sama.
		var countManager int64
		err = t.db.Model(&domain.Manager{}).
			Where("user_id = ?", user.ID).                                         // Filter by user_id
			Joins("JOIN task_managers ON task_managers.manager_id = managers.id"). // Join with task_managers
			Where("task_managers.task_id = ?", task.ID).                           // Filter by task_id
			Count(&countManager).Error
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}
		if countManager > 0 {
			return nil, nil, nil, nil, nil, errors.New("User is already assigned as manager to a task")
		} else {
			// jika kedua validasi tersebut berhasil masukkan data ke table penghubung
			employee.UserID = user.ID
			if err := t.db.Save(employee).Error; err != nil {
				return nil, nil, nil, nil, nil, errors.New("Failed to save manager data")
			}
			sqlQuery := "INSERT INTO task_employees (task_id, employee_id) VALUES (?, ?)"
			if err := t.db.Exec(sqlQuery, task.ID, employee.ID).Error; err != nil {
				return nil, nil, nil, nil, nil, err
			}
		}
	}

	// Simpan planningFile
	if planningFile != nil && (planningFile.FileUrl != "" || planningFile.FileName != "") {
		var count int64
		if err := t.db.Model(&domain.PlanningFile{}).Where("file_url", planningFile.FileUrl).Count(&count).Error; err != nil {
			return nil, nil, nil, nil, nil, err
		}
		if count > 0 {
			return nil, nil, nil, nil, nil, errors.New("File already exist")
		}
		if count == 0 {
			if err := t.db.Save(planningFile).Error; err != nil {
				return nil, nil, nil, nil, nil, fmt.Errorf("Failed to upload file: %v", err)
			}
			// Eksekusi query SQL untuk menambahkan relasi task_project_files
			sqlQuery := "INSERT INTO task_planning_files (task_id, planning_file_id) VALUES (?, ?)"
			if err := t.db.Exec(sqlQuery, task.ID, planningFile.ID).Error; err != nil {
				return nil, nil, nil, nil, nil, err
			}
		}
	}

	// Simpan projectFile
	if projectFile != nil && (projectFile.FileUrl != "" || projectFile.FileName != "") {
		var count int64
		if err := t.db.Model(&domain.ProjectFile{}).Where("file_url", projectFile.FileUrl).Count(&count).Error; err != nil {
			return nil, nil, nil, nil, nil, err
		}
		if count > 0 {
			return nil, nil, nil, nil, nil, errors.New("File already exist")
		}
		if count == 0 {
			if err := t.db.Save(projectFile).Error; err != nil {
				return nil, nil, nil, nil, nil, fmt.Errorf("Failed to upload file: %v", err)
			}
			// Eksekusi query SQL untuk menambahkan relasi task_project_files
			sqlQuery := "INSERT INTO task_project_files (task_id, project_file_id) VALUES (?, ?)"
			if err := t.db.Exec(sqlQuery, task.ID, projectFile.ID).Error; err != nil {
				return nil, nil, nil, nil, nil, err
			}
		}
	}

	return task, manager, employee, planningFile, projectFile, nil
}

func (t *taskAndOwnerRepository) UpdateValidationOwner(taskID uint, userID uint) error {
	task, err := t.FindById(taskID)
	if err != nil {
		return err
	}

	var count int64
	if err := t.db.Model(&domain.Owner{}).
		Where("user_id = ? AND id = ?", userID, task.OwnerID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("Gagal validasi kepemilikan: %v", err)
	}
	if count == 0 {
		return errors.New("Only for owner")
	}

	return nil
}

func (t *taskAndOwnerRepository) UpdateValidationManager(taskID uint, userID uint) error {
	var count int64
	err := t.db.Model(&domain.Manager{}).
		Where("user_id = ?", userID).                                          // Filter by user_id
		Joins("JOIN task_managers ON task_managers.manager_id = managers.id"). // Join with task_managers
		Where("task_managers.task_id = ?", taskID).                            // Filter by task_id
		Count(&count).Error
	if err != nil {
		return fmt.Errorf("Failed to validate manager: %v", err)
	}
	if count == 0 {
		return errors.New("Only for manager")
	}

	return nil
}

func (t *taskAndOwnerRepository) UpdateValidationEmployee(taskID uint, userID uint) error {
	var count int64
	err := t.db.Model(&domain.Employee{}).
		Where("user_id = ?", userID).                                              // Filter by user_id
		Joins("JOIN task_employees ON task_employees.employee_id = employees.id"). // Join with task_employees
		Where("task_employees.task_id = ?", taskID).                               // Filter by task_id
		Count(&count).Error
	if err != nil {
		return fmt.Errorf("Failed to validate employee: %v", err)
	}
	if count == 0 {
		return errors.New("Only for employee")
	}
	return nil
}

func (t *taskAndOwnerRepository) DeleteManager(taskId uint, managerId uint) (*gorm.DB, int64, int64, int64, error) {
	var manager domain.Manager
	if err := t.db.First(&manager, managerId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, 0, 0, errors.New("manager not found")
		}
		return nil, 0, 0, 0, fmt.Errorf("failed to find manager: %v", err)
	}

	sqlQuery := "DELETE FROM task_managers WHERE manager_id = ?"
	if err := t.db.Exec(sqlQuery, manager.ID).Error; err != nil {
		return nil, 0, 0, 0, err
	}

	if err := t.db.Delete(&manager).Error; err != nil {
		return nil, 0, 0, 0, fmt.Errorf("failed to delete manager: %v", err)
	}

	// Periksa apakah ada manager tersisa untuk task
	var count int64
	if err := t.db.Model(&domain.Manager{}).
		Joins("JOIN task_managers on task_managers.manager_id = managers.id").
		Where("task_managers.task_id = ?", taskId).
		Count(&count).Error; err != nil {
		return nil, 0, 0, 0, err
	}

	var (
		countEmployee     int64
		countPlanningFile int64
		countProjectFile  int64
	)

	if count == 0 {
		// jika tidak ada manager tersisa, hapus semua data employee pada task
		var taskEmployeeIDs []uint64
		// menggunakan taskEmployeesQuery.Table untuk count, karena tidak bisa pluck employee_id, dikarenakan tidak ada model dari query ini
		rows, err := t.db.Raw("SELECT employee_id FROM task_employees WHERE task_id = ?", taskId).Rows()
		if err != nil {
			return nil, 0, 0, 0, fmt.Errorf("failed to retrieve task employee IDs: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var employeeID uint64
			if err := rows.Scan(&employeeID); err != nil {
				return nil, 0, 0, 0, fmt.Errorf("failed to read project file ID: %v", err)
			}
			taskEmployeeIDs = append(taskEmployeeIDs, employeeID)
		}

		if len(taskEmployeeIDs) > 0 {
			taskEmployeesDelete := "DELETE FROM task_employees WHERE task_id = ?"
			if err := t.db.Exec(taskEmployeesDelete, taskId).Error; err != nil {
				return nil, 0, 0, 0, err
			}

			employeesDelete := "DELETE FROM employees WHERE id IN (?)"
			if err := t.db.Exec(employeesDelete, taskEmployeeIDs).Error; err != nil {
				return nil, 0, 0, 0, err
			}
			countEmployee = 1
		}

		// jika tidak ada manager tersisa, hapus semua data planning file pada task
		var taskPlanningFileIDs []uint64
		// Eksekusi query untuk mengambil planning_file_id
		rows, err = t.db.Raw("SELECT planning_file_id FROM task_planning_files WHERE task_id = ?", taskId).Rows()
		if err != nil {
			return nil, 0, 0, 0, fmt.Errorf("failed to retrieve task planning files IDs: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var planningFileID uint64
			if err := rows.Scan(&planningFileID); err != nil {
				return nil, 0, 0, 0, fmt.Errorf("failed to read project file ID: %v", err)
			}
			taskPlanningFileIDs = append(taskPlanningFileIDs, planningFileID)
		}

		if len(taskPlanningFileIDs) > 0 {
			taskPlanningFileDelete := "DELETE FROM task_planning_files WHERE task_id = ?"
			if err := t.db.Exec(taskPlanningFileDelete, taskId).Error; err != nil {
				return nil, 0, 0, 0, err
			}

			planningFileDelete := "DELETE FROM planning_files WHERE id IN (?)"
			if err := t.db.Exec(planningFileDelete, taskPlanningFileIDs).Error; err != nil {
				return nil, 0, 0, 0, err
			}
			countPlanningFile = 1
		}

		// Jika tidak ada employee tersisa, hapus semua data project file pada task
		var taskProjectFileIDs []uint64
		// Eksekusi query untuk mengambil project_file_id
		rows, err = t.db.Raw("SELECT project_file_id FROM task_project_files WHERE task_id = ?", taskId).Rows()
		if err != nil {
			return nil, 0, 0, 0, fmt.Errorf("failed to retrieve task project files IDs: %v", err)
		}
		defer rows.Close() // Pastikan rows ditutup dengan benar

		// looping hasil query dalam rows yang datanya adalah nilai actual dari project_file_id, kemdian simpan ke dalam projectFileID
		for rows.Next() {
			var projectFileID uint64
			if err := rows.Scan(&projectFileID); err != nil {
				return nil, 0, 0, 0, fmt.Errorf("failed to read project file ID: %v", err)
			}
			taskProjectFileIDs = append(taskProjectFileIDs, projectFileID)
		}

		if len(taskProjectFileIDs) > 0 {
			// Hapus task_project_files jika ada
			taskProjectFileDelete := "DELETE FROM task_project_files WHERE task_id = ?"
			if err := t.db.Exec(taskProjectFileDelete, taskId).Error; err != nil {
				return nil, 0, 0, 0, err
			}

			// Hapus project_files menggunakan klausa IN
			projectFileDelete := "DELETE FROM project_files WHERE id IN (?)"
			if err := t.db.Exec(projectFileDelete, taskProjectFileIDs).Error; err != nil {
				return nil, 0, 0, 0, err
			}
			countProjectFile = 1
		}
	}

	return t.db, countEmployee, countPlanningFile, countProjectFile, nil
}

func (t *taskAndOwnerRepository) DeleteEmployee(taskId uint, employeeId uint) (*gorm.DB, int64, error) {
	var employee domain.Employee
	if err := t.db.First(&employee, employeeId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, errors.New("employee not found")
		}
		return nil, 0, fmt.Errorf("failed to find employee: %v", err)
	}

	sqlQuery := "DELETE FROM task_employees WHERE employee_id = ?"
	if err := t.db.Exec(sqlQuery, employee.ID).Error; err != nil {
		return nil, 0, err
	}

	if err := t.db.Delete(&employee).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to delete employee: %v", err)
	}

	// Periksa apakah ada employee tersisa untuk task
	var count int64
	if err := t.db.Model(&domain.Employee{}).
		Joins("JOIN task_employees on task_employees.employee_id = employees.id").
		Where("task_employees.task_id = ?", taskId).
		Count(&count).Error; err != nil {
		return nil, 0, err
	}

	var (
		countProjectFile int64
	)

	if count == 0 {
		// Jika tidak ada employee tersisa, hapus semua data project file pada task
		var taskProjectFileIDs []uint64

		// Eksekusi query untuk mengambil project_file_id
		rows, err := t.db.Raw("SELECT project_file_id FROM task_project_files WHERE task_id = ?", taskId).Rows()
		if err != nil {
			return nil, 0, fmt.Errorf("failed to retrieve task project files IDs: %v", err)
		}
		defer rows.Close() // Pastikan rows ditutup dengan benar

		// looping hasil query dalam rows yang datanya adalah nilai actual dari project_file_id, kemdian simpan ke dalam projectFileID
		for rows.Next() {
			var projectFileID uint64
			if err := rows.Scan(&projectFileID); err != nil {
				return nil, 0, fmt.Errorf("failed to read project file ID: %v", err)
			}
			taskProjectFileIDs = append(taskProjectFileIDs, projectFileID)
		}

		if len(taskProjectFileIDs) > 0 {
			// Hapus task_project_files jika ada
			taskProjectFileDelete := "DELETE FROM task_project_files WHERE task_id = ?"
			if err := t.db.Exec(taskProjectFileDelete, taskId).Error; err != nil {
				return nil, 0, err
			}

			// Hapus project_files menggunakan klausa IN
			projectFileDelete := "DELETE FROM project_files WHERE id IN (?)"
			if err := t.db.Exec(projectFileDelete, taskProjectFileIDs).Error; err != nil {
				return nil, 0, err
			}
			countProjectFile = 1
		}
	}

	return t.db, countProjectFile, nil
}

func (t *taskAndOwnerRepository) DeletePlanningFile(fileId uint) (*gorm.DB, string, error) {
	var planningFile domain.PlanningFile
	if err := t.db.First(&planningFile, fileId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", errors.New("file not found")
		}
		return nil, "", fmt.Errorf("failed to find file: %v", err)
	}

	// mengambil file name
	var fileName string
	fileName = planningFile.FileName

	sqlQuery := "DELETE FROM task_planning_files WHERE planning_file_id = ?"
	if err := t.db.Exec(sqlQuery, planningFile.ID).Error; err != nil {
		return nil, "", err
	}

	if err := t.db.Delete(&planningFile).Error; err != nil {
		return nil, "", fmt.Errorf("failed to delete file: %v", err)
	}

	return t.db, fileName, nil
}

func (t *taskAndOwnerRepository) DeleteProjectFile(fileId uint) (*gorm.DB, string, error) {
	var projectFile domain.ProjectFile
	if err := t.db.First(&projectFile, fileId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", errors.New("file not found")
		}
		return nil, "", fmt.Errorf("failed to find file: %v", err)
	}

	var fileName string
	fileName = projectFile.FileName

	sqlQuery := "DELETE FROM task_project_files WHERE project_file_id = ?"
	if err := t.db.Exec(sqlQuery, projectFile.ID).Error; err != nil {
		return nil, "", err
	}

	if err := t.db.Delete(&projectFile).Error; err != nil {
		return nil, "", fmt.Errorf("failed to delete file: %v", err)
	}

	return t.db, fileName, nil
}

func (t *taskAndOwnerRepository) Delete(taskID uint) (*gorm.DB, int64, int64, int64, int64, int64, error) {
	// validasi task
	var task domain.Task
	if err := t.db.First(&task, taskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, 0, 0, 0, 0, errors.New("task not found")
		}
		return nil, 0, 0, 0, 0, 0, fmt.Errorf("failed to find task: %v", err)
	}

	var (
		countOwners       int64
		countManager      int64
		countEmployee     int64
		countPlanningFile int64
		countProjectFile  int64
	)

	// Validasi owners
	var ownerIDs []uint64
	rows, err := t.db.Table("tasks").Select("tasks.owner_id").Joins("INNER JOIN owners ON owners.id = tasks.owner_id").Where("tasks.id = ?", taskID).Rows()
	if err != nil {
		return nil, 0, 0, 0, 0, 0, fmt.Errorf("gagal mengambil ID pemilik untuk task: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ownerID uint64
		if err := rows.Scan(&ownerID); err != nil {
			return nil, 0, 0, 0, 0, 0, fmt.Errorf("gagal membaca ID pemilik: %v", err)
		}
		log.Println(ownerID)
		ownerIDs = append(ownerIDs, ownerID)
	}

	if len(ownerIDs) > 0 {
		if err := t.db.Where("owner_id IN (?)", ownerIDs).Delete(&domain.Task{}).Error; err != nil {
			return nil, 0, 0, 0, 0, 0, fmt.Errorf("gagal menghapus referensi pemilik dari tasks: %v", err)
		}

		if err := t.db.Where("id IN (?)", ownerIDs).Delete(&domain.Owner{}).Error; err != nil {
			return nil, 0, 0, 0, 0, 0, fmt.Errorf("gagal menghapus pemilik dari basis data: %v", err)
		}
		countOwners = 1
	}

	// Validasi manager
	var taskManagerIDs []uint
	// menggunakan taskmanagersQuery.Table untuk count, karena tidak bisa pluck manager_id, dikarenakan tidak ada model dari query ini
	rows, err = t.db.Raw("SELECT manager_id FROM task_managers WHERE task_id = ?", taskID).Rows()
	if err != nil {
		return nil, 0, 0, 0, 0, 0, fmt.Errorf("failed to retrieve task manager IDs: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var managerID uint64
		if err := rows.Scan(&managerID); err != nil {
			return nil, 0, 0, 0, 0, 0, fmt.Errorf("failed to read manager ID: %v", err)
		}
		taskManagerIDs = append(taskManagerIDs, uint(managerID))
	}

	if len(taskManagerIDs) > 0 {
		taskmanagersDelete := "DELETE FROM task_managers WHERE task_id = ?"
		if err := t.db.Exec(taskmanagersDelete, taskID).Error; err != nil {
			return nil, 0, 0, 0, 0, 0, err
		}

		managersDelete := "DELETE FROM managers WHERE id IN (?)"
		if err := t.db.Exec(managersDelete, taskManagerIDs).Error; err != nil {
			return nil, 0, 0, 0, 0, 0, err
		}
		countManager = 1
	}

	// Validasi employee
	var taskEmployeeIDs []uint
	// menggunakan taskEmployeesQuery.Table untuk count, karena tidak bisa pluck employee_id, dikarenakan tidak ada model dari query ini
	rows, err = t.db.Raw("SELECT employee_id FROM task_employees WHERE task_id = ?", taskID).Rows()
	if err != nil {
		return nil, 0, 0, 0, 0, 0, fmt.Errorf("failed to retrieve task employee IDs: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var employeeID uint64
		if err := rows.Scan(&employeeID); err != nil {
			return nil, 0, 0, 0, 0, 0, fmt.Errorf("failed to read project file ID: %v", err)
		}
		taskEmployeeIDs = append(taskEmployeeIDs, uint(employeeID))
	}

	if len(taskEmployeeIDs) > 0 {
		taskEmployeesDelete := "DELETE FROM task_employees WHERE task_id = ?"
		if err := t.db.Exec(taskEmployeesDelete, taskID).Error; err != nil {
			return nil, 0, 0, 0, 0, 0, err
		}

		employeesDelete := "DELETE FROM employees WHERE id IN (?)"
		if err := t.db.Exec(employeesDelete, taskEmployeeIDs).Error; err != nil {
			return nil, 0, 0, 0, 0, 0, err
		}
		countEmployee = 1
	}

	// validasi PlanningFile
	var taskPlanningFileIDs []uint
	// Eksekusi query untuk mengambil planning_file_id
	rows, err = t.db.Raw("SELECT planning_file_id FROM task_planning_files WHERE task_id = ?", taskID).Rows()
	if err != nil {
		return nil, 0, 0, 0, 0, 0, fmt.Errorf("failed to retrieve task planning files IDs: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var planningFileID uint64
		if err := rows.Scan(&planningFileID); err != nil {
			return nil, 0, 0, 0, 0, 0, fmt.Errorf("failed to read project file ID: %v", err)
		}
		taskPlanningFileIDs = append(taskPlanningFileIDs, uint(planningFileID))
	}

	if len(taskPlanningFileIDs) > 0 {
		taskPlanningFileDelete := "DELETE FROM task_planning_files WHERE task_id = ?"
		if err := t.db.Exec(taskPlanningFileDelete, taskID).Error; err != nil {
			return nil, 0, 0, 0, 0, 0, err
		}

		planningFileDelete := "DELETE FROM planning_files WHERE id IN (?)"
		if err := t.db.Exec(planningFileDelete, taskPlanningFileIDs).Error; err != nil {
			return nil, 0, 0, 0, 0, 0, err
		}
		countPlanningFile = 1
	}

	// validasi ProjectFile
	var taskProjectFileIDs []uint
	// Eksekusi query untuk mengambil project_file_id
	rows, err = t.db.Raw("SELECT project_file_id FROM task_project_files WHERE task_id = ?", taskID).Rows()
	if err != nil {
		return nil, 0, 0, 0, 0, 0, fmt.Errorf("failed to retrieve task project files IDs: %v", err)
	}
	defer rows.Close() // Pastikan rows ditutup dengan benar

	// looping hasil query dalam rows yang datanya adalah nilai actual dari project_file_id, kemdian simpan ke dalam projectFileID
	for rows.Next() {
		var projectFileID uint64
		if err := rows.Scan(&projectFileID); err != nil {
			return nil, 0, 0, 0, 0, 0, fmt.Errorf("failed to read project file ID: %v", err)
		}
		taskProjectFileIDs = append(taskProjectFileIDs, uint(projectFileID))
	}

	if len(taskProjectFileIDs) > 0 {
		// Hapus task_project_files jika ada
		taskProjectFileDelete := "DELETE FROM task_project_files WHERE task_id = ?"
		if err := t.db.Exec(taskProjectFileDelete, taskID).Error; err != nil {
			return nil, 0, 0, 0, 0, 0, err
		}

		// Hapus project_files menggunakan klausa IN
		projectFileDelete := "DELETE FROM project_files WHERE id IN (?)"
		if err := t.db.Exec(projectFileDelete, taskProjectFileIDs).Error; err != nil {
			return nil, 0, 0, 0, 0, 0, err
		}
		countProjectFile = 1
	}

	// hapus entri dari task berdasarkan id yang ditemukan
	if err := t.db.Delete(&task).Error; err != nil {
		return nil, 0, 0, 0, 0, 0, fmt.Errorf("failed to delete tasks: %v", err)
	}

	return t.db, countOwners, countManager, countEmployee, countPlanningFile, countProjectFile, nil
}
