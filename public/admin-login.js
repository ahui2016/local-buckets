$("title").text("Admin Login (管理員登入) - Local Buckets");

const navBar = m("div")
  .addClass("row")
  .append(
    m("div")
      .addClass("col text-start")
      .append(
        MJBS.createLinkElem("index.html", { text: "Home" }),
        span(" .. Admin Login (管理員登入)")
      ),
    m("div")
      .addClass("col text-end")
      .append(
        MJBS.createLinkElem("/files.html", { text: "Files" }),
        " | ",
        MJBS.createLinkElem("#", { text: "Link2" }).addClass("Link2")
      )
  );

const PageAlert = MJBS.createAlert();
const PageLoading = MJBS.createLoading(null, "large");

const LogoutBtn = MJBS.createButton("Logout", "warning");
const LogoutBtnArea = cc("div", {
  classes: "text-center",
  children: [
    m(LogoutBtn).on("click", (event) => {
      event.preventDefault();
      MJBS.disable(LogoutBtn);
      axiosGet({
        url: "/api/logout",
        alert: PageAlert,
        onSuccess: () => {
          PageAlert.clear().insert("warning", "已登出");
          LogoutBtnArea.hide();
          LoginForm.show();
          MJBS.focus(PasswordInput);
        },
        onAlways: () => {
          MJBS.enable(LogoutBtn);
        },
      });
    }),
  ],
});

const PasswordInput = MJBS.createInput("password", "required");
const LoginBtn = MJBS.createButton("Login", "primary", "submit");

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
          PasswordInput.setVal("");
          LoginForm.hide();
          LogoutBtnArea.show();
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
  .css(RootCss)
  .append(
    navBar.addClass("my-3"),
    m(PageAlert).addClass("my-5"),
    m(PageLoading).addClass("my-5"),
    m(LoginForm).addClass("my-5").hide(),
    m(LogoutBtnArea).addClass("my-5").hide()
  );

init();

function init() {
  getLoginStatus();
}

function getLoginStatus() {
  axiosGet({
    url: "/api/login-status",
    alert: PageAlert,
    onSuccess: (resp) => {
      const loginStatus = resp.data;
      if (loginStatus.text == "logged-in") {
        PageAlert.clear().insert("light", "已登入");
        LogoutBtnArea.show();
      } else {
        LoginForm.show();
        MJBS.focus(PasswordInput);
      }
    },
    onAlways: () => {
      PageLoading.hide();
    },
  });
}
