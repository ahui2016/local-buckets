$("title").text("Edit Bucket Attributes (修改倉庫屬性) - Local Buckets");

const navBar = m("div")
  .addClass("row")
  .append(
    m("div")
      .addClass("col text-start")
      .append(
        MJBS.createLinkElem("index.html", { text: "Home" }),
        span(" .. 修改倉庫屬性")
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

const IdInput = MJBS.createInput("number", "required"); // readonly
const NameInput = MJBS.createInput("text", "required");
const TitleInput = MJBS.createInput();
const SubtitleInput = MJBS.createInput();
const EncryptedInput = MJBS.createInput(); // readonly

const SubmitBtn = MJBS.createButton("Submit");
const SubmitBtnAlert = MJBS.createAlert();

const EditBucketForm = cc("form", {
  attr: { autocomplete: "off" },
  children: [
    MJBS.hiddenButtonElem(),

    MJBS.createFormControl(IdInput, "ID"),
    MJBS.createFormControl(
      NameInput,
      "Name",
      "倉庫資料夾名稱, 只能使用 0-9, a-z, A-Z, _(下劃線), -(連字號), .(點)"
    ),
    MJBS.createFormControl(TitleInput, "Title"),
    MJBS.createFormControl(SubtitleInput, "Subtitle"),
    MJBS.createFormControl(EncryptedInput, "Encrypted"),

    m(SubmitBtnAlert).addClass("my-3"),
    m("div")
      .addClass("text-center my-3")
      .append(
        m(SubmitBtn).on("click", (event) => {
          event.preventDefault();

          const body = {
            id: IdInput.intVal(),
            name: NameInput.val(),
            title: TitleInput.val(),
            subtitle: SubtitleInput.val(),
          };

          MJBS.disable(SubmitBtn);
          axiosPost({
            url: "/api/update-bucket-info",
            alert: SubmitBtnAlert,
            body: body,
            onSuccess: () => {
              SubmitBtnAlert.clear().insert("success", "修改成功");
            },
            onAlways: () => {
              MJBS.enable(SubmitBtn);
            },
          });
        })
      ),
  ],
});

$("#root")
  .css(RootCss)
  .append(
    navBar.addClass("my-3"),
    m(PageAlert).addClass("my-5"),
    m(PageLoading).addClass("my-5"),
    m(EditBucketForm).addClass("my-5").hide()
  );

init();

function init() {
  let bucketID = getUrlParam("id");
  if (!bucketID) {
    PageLoading.hide();
    PageAlert.insert("danger", "id is null");
    return;
  }
  bucketID = parseInt(bucketID);
  initEditBucketForm(bucketID);
}

function initEditBucketForm(bucketID) {
  axiosPost({
    url: "/api/get-bucket",
    alert: PageAlert,
    body: { id: bucketID },
    onSuccess: (resp) => {
      const bucket = resp.data;

      IdInput.setVal(bucket.id);
      NameInput.setVal(bucket.name);
      TitleInput.setVal(bucket.title);
      SubtitleInput.setVal(bucket.subtitle);
      EncryptedInput.setVal(bucket.encrypted);

      MJBS.disable(IdInput);
      MJBS.disable(EncryptedInput);

      EditBucketForm.show();
    },
    onAlways: () => {
      PageLoading.hide();
    },
  });
}
