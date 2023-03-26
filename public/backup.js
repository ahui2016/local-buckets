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
      "Backup Project",
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
          BKProjCreateAlert.clear().insert("success", "創建備份專案, 成功!");
        },
        onAlways: () => {
          MJBS.enable(BKProjCreateBtn); // ------------------- enable
        },
      });
    }),
  ],
});

$("#root")
  .css(RootCss)
  .append(
    navBar.addClass("my-3"),
    m(PageAlert).addClass("my-5"),
    m(CreateBKProjForm).addClass("my-5")
  );

init();

function init() {
  MJBS.focus(BKProjPathInput);
}
