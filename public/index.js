const pageTitle = m("h5").text("Local Buckets").addClass("display-5");
const pageSubtitle = m("p")
  .text("本地文件倉庫 (管理文件, 備份文件)")
  .addClass(".lead");
const pageTitleArea = m("div")
  .append(pageTitle, pageSubtitle)
  .addClass("text-center");

const AppAlert = MJBS.createAlert();

const ProjectToasts = MJBS.createToasts(
  "toast-container position-fixed top-50 start-50 translate-middle"
);
$("#root").append(m(ProjectToasts));
const ProjectToast = ProjectToasts.new();
ProjectToast.setTitle("Change Project (切換項目)");

function ProjectListItem(project) {
  const self = cc("li", {
    id: `p-${project.id}`,
    classes: "list-group-item d-flex justify-cotent-between align-items-start",
    children: [
      m("div")
        .addClass("ms-2 me-auto")
        .append(
          m("div").text(project.title).addClass("fs-5 fw-bold ProjectTitle"),
          span(project.subtitle).addClass("ProjectSubtitle"),
          span(project.path).addClass("text-muted ProjectPath")
        ),

      // in use
      span("in use")
        .attr({ title: "當前正在使用中" })
        .addClass("UsingProject badge text-bg-secondary")
        .css({ cursor: "default" })
        .hide(),

      // click to use
      span("click to use")
        .attr({ title: "點擊使用該項目" })
        .addClass("UseProjectBtn badge text-bg-light text-muted")
        .css({ cursor: "pointer" })
        .on("click", (event) => {
          event.preventDefault();
          const btnID = `#p-${project.id} .UseProjectBtn`;
          MJBS.disable(btnID)
          axios
            .post("/api/change-project", { id: project.id })
            .then((resp) => {
              const project2 = resp.data;
              ProjectToast.popup(
                m("p")
                  .addClass("mt-3 mb-5 text-center")
                  .append(
                    span(
                      `成功切換項目至 ${project2.title}`
                    ),
                    m("br"),
                    span("3 秒後自動刷新頁面")
                  ),
                null,
                "success"
              );
              setTimeout(() => {
                window.location.reload();
              }, 3000);
            })
            .catch((err) => {
              MJBS.enable(btnID);
              const errMsg = axiosErrToStr(err, validationErrorData_toString);
              ProjectToast.popup(
                m("p").text(errMsg).addClass("mt-3 mb-5 text-center"),
                "danger"
              );
            });
        })
        .hide(),
    ],
  });

  self.init = () => {
    if (project.in_use) {
      self.elem().find(".UsingProject").show();
    } else {
      self.elem().find(".UseProjectBtn").show();
    }
  };
  return self;
}

const ProjectList = cc("ul", { classes: "list-group" });

const FormAlert_AddProject = MJBS.createAlert();
const ProjectPathInput = MJBS.createInput("text", "required");
const AddProjectBtn = MJBS.createButton("Add", "primary", "submit");

const Form_AddProject = cc("form", {
  attr: { autocomplete: "off" },
  children: [
    m("div")
      .addClass("input-group input-group-lg")
      .append(
        // label
        m("span").addClass("input-group-text").text("Project Path"),

        // text input
        m(ProjectPathInput).addClass("form-control"),

        // submit button
        m(AddProjectBtn).on("click", (event) => {
          event.preventDefault();
          const path = MJBS.valOf(ProjectPathInput, "trim");
          if (!path) {
            FormAlert_AddProject.insert("warning", "必須填寫項目地址");
            MJBS.focus(ProjectPathInput);
            return;
          }
          axiosPost({
            url: "/api/add-project",
            body: { path: path },
            alert: FormAlert_AddProject,
            onSuccess: (resp) => {
              const project = resp.data;
              FormAlert_AddProject.insert(
                "success",
                `成功添加項目 id: ${project.id}, path: ${project.path}`
              );
              FormAlert_AddProject.insert(
                "warning",
                "已自動切換至新項目, 10 秒後自動刷新頁面"
              );
              setTimeout(() => {
                window.location.reload();
              }, 10000);
              ProjectPathInput.elem().val("");
            },
          });
        })
      ),
  ],
});

const FormArea_AddProject = cc("div", {
  children: [m(Form_AddProject), m(FormAlert_AddProject).addClass("my-1")],
});

$("#root")
  .css({ maxWidth: "720px" })
  .append(
    pageTitleArea.addClass("my-5"),
    m(AppAlert).addClass("my-3"),
    m(FormArea_AddProject).addClass("my-3"),
    m(ProjectList).addClass("my-5")
  );

init();

function init() {
  // initProjects();
}

function initProjects() {
  axiosGet({
    url: "/api/all-projects",
    alert: AppAlert,
    onSuccess: (resp) => {
      const projects = resp.data;
      if (projects && projects.length > 0) {
        MJBS.appendToList(ProjectList, projects.map(ProjectListItem));
      } else {
        AppAlert.insert("info", "尚未註冊項目, 請添加項目.");
        MJBS.focus(ProjectPathInput);
      }
    },
  });
}
