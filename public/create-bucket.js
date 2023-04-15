$("title").text("Create a Bucket (新建倉庫) - Local Buckets");

const navBar = m("div")
  .addClass("row")
  .append(
    m("div")
      .addClass("col text-start")
      .append(
        MJBS.createLinkElem("index.html", { text: "Local-Buckets" }),
        span(" .. Create a Bucket (新建倉庫)")
      ),
    m("div")
      .addClass("col text-end")
      .append(MJBS.createLinkElem("/buckets.html", { text: "Buckets" }))
  );

const PageAlert = MJBS.createAlert();

const BucketNameInput = MJBS.createInput("text", "required");
const BucketEncryptBox = MJBS.createInput("checkbox");
const CreateBucketBtn = MJBS.createButton("Create");

const CreateBucketForm = cc("form", {
  attr: { autocomplete: "off" },
  children: [
    MJBS.createFormControl(
      BucketNameInput,
      "Bucket Name",
      "倉庫資料夾名稱, 只能使用 0-9, a-z, A-Z, _(下劃線), -(連字號), .(點)"
    ),
    MJBS.createFormCheck(BucketEncryptBox, "Secret Bucket", "設為加密倉庫"),
    MJBS.hiddenButtonElem(),
    m(CreateBucketBtn).on("click", (event) => {
      event.preventDefault();
      const bucketName = BucketNameInput.val();
      const encrypted = BucketEncryptBox.isChecked();
      if (!bucketName) {
        PageAlert.insert("warning", "請填寫 Bucket Name");
        return;
      }
      MJBS.disable(CreateBucketBtn); // --------------------- disable
      axiosPost({
        url: "/api/create-bucket",
        body: {
          name: bucketName,
          encrypted: encrypted,
        },
        alert: PageAlert,
        onSuccess: (resp) => {
          const bucket = resp.data;
          // window.location.href = `edit-bucket.html?id=${bucket.id}&new=true`;
          PageAlert.insert("success", JSON.stringify(bucket));
          CreateBucketBtn.hide();
        },
        onAlways: () => {
          MJBS.enable(CreateBucketBtn); // --------------------- enable
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
    m(CreateBucketForm).addClass("my-5")
  );

init();

function init() {
  MJBS.focus(BucketNameInput);
}
