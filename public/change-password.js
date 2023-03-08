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

const OldPassword = MJBS.createInput();
const CheckPwdBtn = MJBS.createButton("Check", 'primary', 'submit');

const ChangePwdForm = cc("form", {
  attr: { autocomplete: "off" },
  children: [
    MJBS.createFormControl(OldPassword, "Old Password", "舊密碼, 原密碼"),
    m(CheckPwdBtn).on("click", (event) => {
      event.preventDefault();
      axiosPost({
        url: "/api/check-password",
        body: {old_password: MJBS.valOf(OldPassword)},
        alert: PageAlert,
        onSuccess: resp => {
          PageAlert.insert('success', '密碼正確');
        }
      });
    }),
  ],
});

$("#root")
  .css({ maxWidth: "992px" })
  .append(
    navBar.addClass("my-5"),
    m(PageAlert).addClass("my-3"),
    m(ChangePwdForm).addClass("my-3")
  );

init();

function init() {}
