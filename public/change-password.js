$("title").text("Change Password (更改密碼) - Local Buckets");

const navBar = m("div")
  .addClass("row")
  .append(
    m("div")
      .addClass("col text-start")
      .append(
        MJBS.createLinkElem("index.html", { text: "Local-Buckets" }),
        span(" .. Change Password (更改密碼)")
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

const OldPasswordInput = MJBS.createInput("password", "required");
const NewPasswordInput = MJBS.createInput("password", "required");
const ConfirmPwdInput = MJBS.createInput("password", "required");
const ChangePwdBtn = MJBS.createButton("Submit", "primary", "submit");

const ChangePwdForm = cc("form", {
  attr: { autocomplete: "off" },
  children: [
    MJBS.createFormControl(
      OldPasswordInput,
      "Old Password",
      "舊密碼 (當前密碼)"
    ),
    MJBS.createFormControl(NewPasswordInput, "New Password", "新密碼"),
    MJBS.createFormControl(
      ConfirmPwdInput,
      "Confirm New Password",
      "再輸入一次新密碼"
    ),
    MJBS.hiddenButtonElem(),
    m(ChangePwdBtn).on("click", (event) => {
      event.preventDefault();
      const oldPwd = OldPasswordInput.val();
      const newPwd = NewPasswordInput.val();
      const newPwd2 = ConfirmPwdInput.val();
      if (oldPwd == "" || newPwd == "") {
        PageAlert.insert("warning", "請填寫舊密碼和新密碼");
        return;
      }
      if (newPwd != newPwd2) {
        PageAlert.insert("warning", "兩次輸入新密碼必須相同");
        return;
      }
      MJBS.disable(ChangePwdBtn);  // --------------------- disable
      axiosPost({
        url: "/api/change-password",
        body: {
          old_password: oldPwd,
          new_password: newPwd,
        },
        alert: PageAlert,
        onSuccess: () => {
          OldPasswordInput.setVal('');
          NewPasswordInput.setVal('');
          ConfirmPwdInput.setVal('');
          ChangePwdBtn.hide();
          PageAlert.clear().insert("success", "已成功更換密碼");
        },
        onAlways: () => {
          MJBS.enable(ChangePwdBtn);  // ------------------- enable
        }
      });
    }),
  ],
});

$("#root")
  .css(RootCss)
  .append(
    navBar.addClass("my-3"),
    m(PageAlert).addClass("my-5"),
    m(ChangePwdForm).addClass("my-5")
  );

init();

function init() {
  PageAlert.insert(
    "primary",
    "提醒: 請記住新密碼, 一旦忘記將無法解密. (初始密碼: abc123)",
    "no-time"
  );
  PageAlert.insert(
    "info",
    "在更改密碼前, 建議先備份密鑰 (在 project.toml 內)",
    "no-time"
  );
  MJBS.focus(OldPasswordInput);
}
