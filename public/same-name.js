const SameNameRadioName = "SameNameRadio";

const OverwriteRadio = cc("input", {
  classes: "form-check-input",
  type: "radio",
  attr: { type: "radio", name: SameNameRadioName, value: "overwrite" },
});

const OverwriteRadioArea = cc("div", {
  classes: "form-check",
  children: [
    m(OverwriteRadio),
    m("label")
      .addClass("form-check-label")
      .attr({ for: OverwriteRadio.raw_id })
      .text("Overwrite"),
  ],
});

const RenameRadio = cc("input", {
  classes: "form-check-input",
  type: "radio",
  attr: { type: "radio", name: SameNameRadioName, value: "rename" },
});

const RenameRadioArea = cc("div", {
  classes: "form-check",
  children: [
    m(RenameRadio),
    m("label")
      .addClass("form-check-label")
      .attr({ for: RenameRadio.raw_id })
      .text("Rename"),
  ],
});

const SameNameRadioCard = cc("div", {
  classes: "card",
  children: [
    m("div").addClass('card-body').append(
      m(OverwriteRadioArea),
      m(RenameRadioArea),        
    )
  ]
});

SameNameRadioCard.init = () => {
  $(`input[name="${SameNameRadioName}"]`).on("change", () => {
    const val = getSameNameRadioValue();
    console.log(val);
  });
};

function getSameNameRadioValue() {
  return $(`input[name="${SameNameRadioName}"]:checked`).val();
}
