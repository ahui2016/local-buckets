$("title").text("Backup (備份專案) - Local Buckets");

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
        MJBS.createLinkElem("#", { text: "Link2" }).addClass("Link2")
      )
  );

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

const BKProjList = cc("div", { classes: "accordion" });

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
              m(UseBtn).addClass("me-2"),
              m(DelBtn).on("click", (event) => {
                event.preventDefault();
                ItemAlert.insert(
                  "warning",
                  "待 delete 按鈕變紅色後再點擊一次, 執行刪除."
                );
                setTimeout(() => {
                  DelBtn.hide();
                  DangerDelBtn.show();
                }, 5000);
              }),
              m(DangerDelBtn).hide()
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

$("#root")
  .css(RootCss)
  .append(
    navBar.addClass("my-3"),
    m(PageAlert).addClass("my-5"),
    m(PageLoading).addClass("my-5"),
    m(CreateBKProjForm).addClass("my-5").hide(),
    m(BKProjList).addClass("my-5").hide()
  );

init();

function init() {
  getBKProjects();
  MJBS.focus(BKProjPathInput);
}

function getBKProjects() {
  axiosGet({
    url: "/api/project-config",
    alert: PageAlert,
    onSuccess: (resp) => {
      const bkProjects = resp.data.backup_projects;
      if (bkProjects && bkProjects.length > 0) {
        BKProjList.show();
        MJBS.appendToList(BKProjList, bkProjects.map(BKProjItem));
      }
    },
    onAlways: () => {
      PageLoading.hide();
      CreateBKProjForm.show();
    },
  });
}
