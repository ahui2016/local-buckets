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

const OldPasswordInput = MJBS.createInput('text', 'required');
const NewPasswordInput = MJBS.createInput('text', 'required');
const ChangePwdBtn = MJBS.createButton("Submit", "primary", "submit");

const ChangePwdForm = cc("form", {
  attr: { autocomplete: "off" },
  children: [
    MJBS.createFormControl(OldPasswordInput, "Old Password", "舊密碼 (當前密碼)"),
    MJBS.createFormControl(NewPasswordInput, "New Password", "新密碼"),
    m(ChangePwdBtn).on("click", (event) => {
      event.preventDefault();
      axiosPost({
        url: "/api/change-password",
        body: {
          old_password: MJBS.valOf(OldPasswordInput),
          new_password: MJBS.valOf(NewPasswordInput)
        },
        alert: PageAlert,
        onSuccess: (resp) => {
          PageAlert.insert("success", "密碼正確");
        },
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
