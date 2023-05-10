const pageTitle = m("h5").text("Local Buckets").addClass("display-5");
const pageSubtitle = m("p")
  .text("本地資料倉庫 (管理資料, 備份資料)")
  .addClass(".lead");
const pageTitleArea = m("div")
  .append(pageTitle, pageSubtitle)
  .addClass("text-center");

const PageAlert = MJBS.createAlert();

const ProjectInfoAlert = MJBS.createAlert();
const ProjectInfoLoading = MJBS.createLoading("", "large");

const ProjectInfo = cc("div", {
  classes: "card",
  children: [
    m("div")
      .addClass("card-header")
      .append(
        span("Backup Project (備份專案)")
          .addClass("BackupProject badge text-bg-dark fs-6")
          .hide(),
        span("ℹ️")
          .addClass("BcakupProject")
          .css({ cursor: "pointer" })
          .on("click", (event) => {
            event.preventDefault();
            ProjectInfoAlert.insert(
              "info",
              "備份專案, 專用於備份, 部分功能不可用.",
              "no-time"
            );
          })
          .hide(),

        span("Project (正在使用的專案)").addClass("NormalProject"),
        span("ℹ️")
          .addClass("NormalProject")
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
        m("div")
          .addClass("row")
          .append(
            m("div")
              .addClass("col-9")
              .append(
                m("div").addClass("ProjectTitle card-title fw-bold mb-0"),
                m("div").addClass("ProjectSubtitle text-muted mt-0"),
                m("div").addClass("ProjectPath text-muted mt-1")
              ),
            m("div")
              .addClass("col-3 text-end text-muted small")
              .append(
                m("div").addClass("ProjectFilesCount"),
                m("div").addClass("ProjectTotalSize")
              )
          )
      ),
  ],
});

ProjectInfo.fill = (project) => {
  const elem = ProjectInfo.elem();
  if (project.is_backup) {
    elem.find(".NormalProject").hide();
    elem.find(".BackupProject").show();
  }
  elem.find(".ProjectTitle").text(project.title);
  elem.find(".ProjectSubtitle").text(project.subtitle);
  elem.find(".ProjectPath").text(project.Root);
  if (project.FilesCount > 1) {
    elem.find(".ProjectFilesCount").text(`${project.FilesCount} files`);
  }
  elem
    .find(".ProjectTotalSize")
    .text(`(${fileSizeToString(project.TotalSize)})`);
};

const LinkList = cc("div", {
  children: [
    createIndexItem("Recent Files", "files.html", "檔案清單"),
    createIndexItem("Recent Pics", "pics.html", "圖片清單"),
    createIndexItem("Upload", "waiting.html", "上傳檔案"),
    createIndexItem("All Buckets", "buckets.html", "倉庫清單"),
    createIndexItem("Create Bucket", "create-bucket.html", "新建倉庫").addClass(
      "HideIfBackup"
    ),
    createIndexItem(
      "Change Password",
      "change-password.html",
      "更改密碼"
    ).addClass("HideIfBackup"),
    createIndexItem("Backup", "backup.html", "備份專案"),
    createIndexItem("Admin Login", "admin-login.html", "管理登入"),
  ],
});

$("#root")
  .css(RootCss)
  .append(
    pageTitleArea.addClass("my-5"),
    m(PageAlert).addClass("my-3"),
    m(ProjectInfo).addClass("my-3"),
    m(LinkList).addClass("my-5").hide()
  );

init();

function init() {
  initProjectInfo();
}

function initProjectInfo() {
  axiosGet({
    url: "/api/project-status",
    alert: PageAlert,
    onSuccess: (resp) => {
      const project = resp.data;
      ProjectInfo.fill(project);
      LinkList.show();
      initBackupProject(project, PageAlert);
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
