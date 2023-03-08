
const navBar = m("div")
  .addClass("row")
  .append(
    m("div")
      .addClass("col text-start")
      .append(
        MJBS.createLinkElem("index.html", { text: "Local-Buckets" }),
        MJBS.span(" .. Create Bucket (新建倉庫)")
      ),
    m("div")
      .addClass("col text-end")
      .append(
        MJBS.createLinkElem("#", { text: "Link1" }).addClass(
          "Link1"
        ),
        " | ",
        MJBS.createLinkElem("#", { text: "Link2" }).addClass(
          "Link2"
        ),
      )
  );

const PageAlert = MJBS.createAlert();

const ProjectInfo = cc("div", {
  classes: "card",
  children: [
    m("div")
      .addClass("card-header")
      .append(
        span("Project (正在使用的項目)"),
        span("ℹ️")
          .css({ cursor: "pointer" })
          .on("click", (event) => {
            event.preventDefault();
            ProjectInfoAlert.insert(
              "info",
              "用文本編輯器打開項目文件夾(資料夾)內的 project.toml, 可更改項目設定. " +
                "注意, 用 utf-8 編碼保存文件. " +
                "需要重啟程式才生效.",
              false
            );
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
