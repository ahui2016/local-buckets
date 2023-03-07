const pageTitle = m("h5").text("Local Buckets").addClass("display-5");
const pageSubtitle = m("p")
  .text("本地文件倉庫 (管理文件, 備份文件)")
  .addClass(".lead");
const pageTitleArea = m("div")
  .append(pageTitle, pageSubtitle)
  .addClass("text-center");

const AppAlert = MJBS.createAlert();

const ProjectInfoAlert = MJBS.createAlert();

const ProjectInfo = cc("div", {
  classes: "card",
  children: [
    m("div").addClass("card-header").append(
      span("Project (正在使用的項目)"),
      span('ℹ️').css({cursor:'pointer'}).on('click', event => {
        event.preventDefault();
        ProjectInfoAlert.insert('info', '用文本編輯器打開項目文件');
      }),
      m(ProjectInfoAlert)
    ),
    m("div")
      .addClass("card-body")
      .append(
        m("div").addClass("Project-Title card-title fw-bold"),
        m("div").addClass("Project-Subtitle text-muted"),
        m("div").addClass("Project-Path text-muted")
      ),
  ],
});

ProjectInfo.fill = (project) => {
  const elem = ProjectInfo.elem();
  elem.find(".Project-Title").text(project.title);
  elem.find(".Project-Subtitle").text(project.subtitle);
  elem.find(".Project-Path").text(project.path);
};

$("#root")
  .css({ maxWidth: "992px" })
  .append(
    pageTitleArea.addClass("my-5"),
    m(AppAlert).addClass("my-3"),
    m(ProjectInfo).addClass("my-3")
  );

init();

function init() {
  initProjectInfo();
}

function initProjectInfo() {
  axiosGet({
    url: "/api/project-config",
    alert: AppAlert,
    onSuccess: (resp) => {
      const project = resp.data;
      ProjectInfo.fill(project);
    },
  });
}
