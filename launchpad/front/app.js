const API_BASE = "http://localhost:8080";

const errorBox = document.getElementById("error");
const projectsBody = document.getElementById("projects-body");
const ownerSelect = document.getElementById("owner_id");
const createForm = document.getElementById("create-form");

let usersById = {};

function showError(message) {
    errorBox.textContent = message;
    errorBox.classList.remove("hidden");
}

function clearError() {
    errorBox.classList.add("hidden");
    errorBox.textContent = "";
}

async function apiFetch(path, options) {
    let res;
    try {
        res = await fetch(API_BASE + path, options);
    } catch (err) {
        throw new Error("сервер недоступен, проверь, что он запущен на порту 8080");
    }

    if (res.status === 204) {
        return null;
    }

    const data = await res.json().catch(() => null);

    if (!res.ok) {
        const message = (data && data.error) || `ошибка ${res.status}`;
        throw new Error(message);
    }

    return data;
}

async function loadUsers() {
    const users = await apiFetch("/users");
    usersById = {};
    ownerSelect.innerHTML = "";
    for (const user of users) {
        usersById[user.id] = user;
        const option = document.createElement("option");
        option.value = user.id;
        option.textContent = `${user.name} (${user.role})`;
        ownerSelect.appendChild(option);
    }
}

function formatBudget(budget) {
    if (budget === null || budget === undefined) {
        return "—";
    }
    return Number(budget).toLocaleString("ru-RU") + " ₽";
}

function ownerName(ownerId) {
    const user = usersById[ownerId];
    return user ? user.name : `#${ownerId}`;
}

function renderProjects(projects) {
    projectsBody.innerHTML = "";

    if (projects.length === 0) {
        projectsBody.innerHTML = '<tr><td colspan="5">Проектов пока нет</td></tr>';
        return;
    }

    for (const project of projects) {
        const row = document.createElement("tr");

        row.innerHTML = `
            <td>${project.title}</td>
            <td><span class="status status-${project.status}">${project.status}</span></td>
            <td>${ownerName(project.owner_id)}</td>
            <td>${formatBudget(project.budget)}</td>
            <td><button class="delete-btn" data-id="${project.id}">Удалить</button></td>
        `;

        row.querySelector(".delete-btn").addEventListener("click", () => deleteProject(project.id));
        projectsBody.appendChild(row);
    }
}

async function loadProjects() {
    const projects = await apiFetch("/projects");
    renderProjects(projects);
}

async function deleteProject(id) {
    if (!confirm("Удалить проект?")) {
        return;
    }
    clearError();
    try {
        await apiFetch(`/projects/${id}`, { method: "DELETE" });
        await loadProjects();
    } catch (err) {
        showError(err.message);
    }
}

createForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    clearError();

    const title = document.getElementById("title").value.trim();
    const description = document.getElementById("description").value.trim();
    const budget = document.getElementById("budget").value;

    const body = {
        title,
        owner_id: Number(ownerSelect.value),
    };
    if (description) {
        body.description = description;
    }
    if (budget) {
        body.budget = Number(budget);
    }

    try {
        await apiFetch("/projects", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(body),
        });
        createForm.reset();
        await loadProjects();
    } catch (err) {
        showError(err.message);
    }
});

async function init() {
    clearError();
    try {
        await loadUsers();
        await loadProjects();
    } catch (err) {
        showError(err.message);
    }
}

init();
