$("title").text("Admin Login (管理員登入) - Local Buckets");

const navBar = m("div")
  .addClass("row")
  .append(
    m("div")
      .addClass("col text-start")
      .append(
        MJBS.createLinkElem("index.html", { text: "Local-Buckets" }),
        span(" .. Admin Login (管理員登入)")
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

const PasswordInput = MJBS.createInput("password", "required");
const LoginBtn = MJBS.createButton("Submit", "primary", "submit");

const LoginForm = cc("form", {
  attr: { autocomplete: "off" },
  children: [
    MJBS.createFormControl(PasswordInput, "Password", "管理員密碼"),
    m(LoginBtn).on("click", (event) => {
      event.preventDefault();
      const pwd = PasswordInput.val();
      if (pwd == "") {
        PageAlert.insert("warning", "請輸入密碼");
        return;
      }
      MJBS.disable(LoginBtn); // --------------------- disable
      axiosPost({
        url: "/api/admin-login",
        body: { text: pwd },
        alert: PageAlert,
        onSuccess: () => {
          PasswordInput.setVal('');
          LoginForm.hide();
          PageAlert.clear().insert("success", "已成功登入");
        },
        onAlways: () => {
          MJBS.enable(LoginBtn); // ------------------- enable
        },
      });
    }),
  ],
});

$("#root")
  .css({ maxWidth: "768px" })
  .append(
    navBar.addClass("my-3"),
    m(PageAlert).addClass("my-5"),
    m(LoginForm).addClass("my-5")
  );

init();

function init() {
  MJBS.focus(PasswordInput);
}
