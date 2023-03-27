$("title").text("Backup (備份專案) - Local Buckets");

let bkProjStat = null;

const PageAlert = MJBS.createAlert();
const PageLoading = MJBS.createLoading(null, "large");

const BKProjPathInput = MJBS.createInput("text", "required");
const BKProjCreateBtn = MJBS.createButton("Create");
const BKProjCreateAlert = MJBS.createAlert();

const CreateBKProjForm = cc("form", {
  attr: { autocomplete: "off" },
  children: [
    MJBS.createFormControl(
      BKProjPathInput,
      "新建備份專案",
      "備份專案的絕對路徑, 必須是一個空資料夾."
    ),
    MJBS.hiddenButtonElem(),
    m(BKProjCreateAlert),
    m(BKProjCreateBtn).on("click", (event) => {
      event.preventDefault();
      const bkProjectPath = BKProjPathInput.val();
      if (!bkProjectPath) {
        BKProjCreateAlert.insert(
          "warning",
          "請填寫 Backup Project 備份專案的絕對路徑"
        );
        MJBS.focus(BKProjPathInput);
        return;
      }
      MJBS.disable(BKProjCreateBtn); // --------------------- disable
      axiosPost({
        url: "/api/create-bk-proj",
        body: { text: bkProjectPath },
        alert: BKProjCreateAlert,
        onSuccess: () => {
          BKProjCreateBtn.hide();
          BKProjCreateAlert.clear().insert("success", "創建備份專案, 成功!");
          BKProjCreateAlert.insert("info", "三秒後將會自動刷新本頁");
          setTimeout(() => {
            window.location.reload();
          }, 5000);
        },
        onAlways: () => {
          MJBS.enable(BKProjCreateBtn); // ------------------- enable
          MJBS.focus(BKProjPathInput);
        },
      });
    }),
  ],
});

const CreateBKProjLinkArea = cc("div", { css: { display: "inline" } });

const navBar = m("div")
  .addClass("row")
  .append(
    m("div")
      .addClass("col text-start")
      .append(
        MJBS.createLinkElem("index.html", { text: "Local-Buckets" }),
        span(" .. Backup (備份專案)")
      ),
    m("div")
      .addClass("col text-end")
      .append(
        MJBS.createLinkElem("#", { text: "Link1" }).addClass("Link1"),
        " | ",
        MJBS.createLinkElem("#", { text: "Link2" }).addClass("Link2"),
        m(CreateBKProjLinkArea).append(
          " | ",
          MJBS.createLinkElem("#", { text: "新建備份專案" })
            .addClass("CreateBKProjLink")
            .on("click", (event) => {
              event.preventDefault();
              MJBS.disable(".CreateBKProjLink");
              CreateBKProjLinkArea.elem().fadeOut(2000);
              CreateBKProjForm.show();
              MJBS.focus(BKProjPathInput);
            })
        )
      )
  );

const BKProjList = cc("div", { classes: "accordion" });
const BKProjListArea = cc("div", {
  children: [m("h4").text("請選擇備份目的地:"), m(BKProjList)],
});

function BKProjItem(projPath) {
  const UseBtn = MJBS.createButton("use");
  const DelBtn = MJBS.createButton("delete", "secondary");
  const DangerDelBtn = MJBS.createButton("DELETE", "danger");
  const ItemAlert = MJBS.createAlert();

  const body = cc("div", {
    classes: "accordion-collapse collapse",
    attr: { "data-bs-parent": BKProjList.id },
    children: [
      m("div")
        .addClass("accordion-body")
        .append(
          m("div")
            .addClass("text-end mb-2")
            .append(
              m(UseBtn)
                .addClass("me-2")
                .on("click", (event) => {
                  event.preventDefault();
                  MJBS.disable(UseBtn);
                  axiosGet({
                    url: "/api/bk-project-status",
                    alert: ItemAlert,
                    onSuccess: (resp) => {
                      bkProjStat = resp.data;
                      const BKProjStat = createProjStat(bkProjStat);
                      ProjectsStatusArea.elem().append(
                        m("div").addClass("my-2 text-center").html(`
                            <!-- https://icons.getbootstrap.com/icons/arrow-down/ -->
                            <svg
                              xmlns="http://www.w3.org/2000/svg"
                              width="3rem"
                              height="3rem"
                              fill="currentColor"
                              class="bi bi-arrow-down"
                              viewBox="0 0 16 16"
                            >
                              <path
                                fill-rule="evenodd"
                                d="M8 1a.5.5 0 0 1 .5.5v11.793l3.146-3.147a.5.5 0 0 1 .708.708l-4 4a.5.5 0 0 1-.708 0l-4-4a.5.5 0 0 1 .708-.708L7.5 13.293V1.5A.5.5 0 0 1 8 1z"
                              />
                            </svg>`),
                        m(BKProjStat)
                      );
                      BKProjListArea.hide();
                    },
                    onAlways: () => {
                      MJBS.enable(UseBtn);
                    },
                  });
                }),
              m(DelBtn).on("click", (event) => {
                event.preventDefault();
                MJBS.disable(DelBtn);
                ItemAlert.insert(
                  "warning",
                  "待 delete 按鈕變紅色後再點擊一次, 執行刪除."
                );
                setTimeout(() => {
                  DelBtn.hide();
                  DangerDelBtn.show();
                }, 5000);
              }),
              m(DangerDelBtn)
                .hide()
                .on("click", (event) => {
                  event.preventDefault();
                  MJBS.disable(UseBtn);
                  MJBS.disable(DangerDelBtn);
                  axiosPost({
                    url: "/api/delete-bk-proj",
                    alert: ItemAlert,
                    body: { text: projPath },
                    onSuccess: () => {
                      UseBtn.hide();
                      DangerDelBtn.hide();
                      ItemAlert.clear().insert(
                        "success",
                        "該備份倉庫已被刪除 (僅從清單中刪除, 資料夾仍保留)"
                      );
                    },
                    onAlways: () => {
                      MJBS.enable(UseBtn);
                      MJBS.enable(DangerDelBtn);
                    },
                  });
                })
            ),
          m(ItemAlert)
        ),
    ],
  });

  return cc("div", {
    classes: "accordion-item",
    children: [
      m("h2")
        .addClass("accordion-header")
        .append(
          m("button")
            .addClass("accordion-button collapsed")
            .text(projPath)
            .attr({
              type: "button",
              "data-bs-toggle": "collapse",
              "data-bs-target": body.id,
              "aria-expanded": false,
              "aria-controls": body.raw_id,
            })
        ),
      m(body),
    ],
  });
}

function createProjStat(projStat) {
  let projType = "Source (源專案)";
  if (projStat.is_backup) projType = "Destination (備份專案)";

  let lastBackup = projStat.last_backup_at.substr(0, 16);
  if (projStat.last_backup_at == "") lastBackup = "Not Yet";

  return cc("div", {
    classes: "card mb-2",
    children: [
      m("div")
        .addClass("card-header")
        .append(
          m("div").text(projType),
          m("div").text(projStat.Path).addClass("text-muted")
        ),
      m("div")
        .addClass("card-body")
        .append(
          m("dl")
            .addClass("row")
            .append(
              m("dt").addClass("col-sm-3").text("上次備份時間: "),
              m("dt")
                .addClass("col-sm-9 text-muted")
                .text(lastBackup),

              m("dt").addClass("col-sm-3").text("占用空間: "),
              m("dt")
                .addClass("col-sm-9 text-muted")
                .text(fileSizeToString(projStat.TotalSize)),

              m("dt").addClass("col-sm-3").text("全部檔案 (個): "),
              m("dt").addClass("col-sm-9 text-muted").text(projStat.FilesCount),

              m("dt").addClass("col-sm-3").text("其中待檢查檔案 (個): "),
              m("dt")
                .addClass("col-sm-9 text-muted")
                .text(projStat.WaitingCheckCount),

              m("dt").addClass("col-sm-3").text("其中損毀檔案 (個): "),
              m("dt")
                .addClass("col-sm-9 text-muted")
                .text(projStat.DamagedCount)
            )
        ),
    ],
  });
}

const ProjectsStatusArea = cc("div");

$("#root")
  .css(RootCss)
  .append(
    navBar.addClass("my-3"),
    m(PageAlert).addClass("my-5"),
    m(PageLoading).addClass("my-5"),
    m(CreateBKProjForm).addClass("my-5").hide(),
    m(ProjectsStatusArea).addClass("my-5"),
    m(BKProjListArea).addClass("my-5").hide()
  );

init();

function init() {
  getMainProject();
}

function getMainProject() {
  axiosGet({
    url: "/api/project-status",
    alert: PageAlert,
    onSuccess: (resp) => {
      const mainProjStat = resp.data;
      initBKProjects(mainProjStat.backup_projects);

      const MainProjStat = createProjStat(mainProjStat);
      ProjectsStatusArea.elem().append(m(MainProjStat));
    },
    onAlways: () => {
      PageLoading.hide();
    },
  });
}

function initBKProjects(projects) {
  if (projects && projects.length > 0) {
    BKProjListArea.show();
    MJBS.appendToList(BKProjList, projects.map(BKProjItem));
  } else {
    CreateBKProjLinkArea.hide();
    CreateBKProjForm.show();
    MJBS.focus(BKProjPathInput);
  }
}