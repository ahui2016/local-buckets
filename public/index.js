const pageTitle = m("h5").text("Local Buckets").addClass("display-5");
const pageSubtitle = m("p")
  .text("本地資料倉庫 (管理資料, 備份資料)")
  .addClass(".lead");
const pageTitleArea = m("div")
  .append(pageTitle, pageSubtitle)
  .addClass("text-center");

const AppAlert = MJBS.createAlert();

const ProjectInfoAlert = MJBS.createAlert();
const ProjectInfoLoading = MJBS.createLoading("", "large");

const ProjectInfo = cc("div", {
  classes: "card",
  children: [
    m("div")
      .addClass("card-header")
      .append(
        span("Project (正在使用的專案)"),
        span("ℹ️")
          .css({ cursor: "pointer" })
          .on("click", (event) => {
            event.preventDefault();
            ProjectInfoAlert.insert(
              "info",
              "用文本編輯器打開專案資料夾內的 project.toml, 可更改專案設定. " +
                "注意, 用 utf-8 編碼保存檔案. " +
                "需要重啟程式才生效.",
              "no-time"
            );
          }),
        m(ProjectInfoAlert)
      ),
    m("div")
      .addClass("card-body")
      .append(
        m(ProjectInfoLoading).addClass("my-3"),
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

const LinkList = cc("div", {
  children: [
    createIndexItem("Recent Files", "recent-files.html", "最近檔案"),
    createIndexItem("Upload", "waiting.html", "上傳檔案"),
    createIndexItem("Create Bucket", "create-bucket.html", "新建倉庫"),
    createIndexItem("Change Password", "change-password.html", "更改密碼"),
    createIndexItem("Backup", "backup.html", "備份專案"),
  ],
});

$("#root")
  .css(RootCss)
  .append(
    pageTitleArea.addClass("my-5"),
    m(AppAlert).addClass("my-3"),
    m(ProjectInfo).addClass("my-3"),
    m(LinkList).addClass("my-5")
  );

init();

function init() {
  initProjectInfo();
}

function initProjectInfo() {
  axiosGet({
    url: "/api/project-info",
    alert: AppAlert,
    onSuccess: (resp) => {
      const project = resp.data;
      ProjectInfo.fill(project);
    },
    onAlways: () => {
      ProjectInfoLoading.hide();
    },
  });
}

function createIndexItem(text, link, description) {
  return m("div")
    .addClass("row mb-2 g-1")
    .append(
      m("div").addClass("col text-end").text(description),
      m("div")
        .addClass("col")
        .append(
          m("a")
            .text(text)
            .attr({ href: link })
            .addClass("text-decoration-none")
        )
    );
}
